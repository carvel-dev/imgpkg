// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imageset

import (
	"fmt"
	"io"
	"os"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
)

type TarImageSet struct {
	imageSet    ImageSet
	concurrency int
	logger      Logger
}

func NewTarImageSet(imageSet ImageSet, concurrency int, logger Logger) TarImageSet {
	return TarImageSet{imageSet, concurrency, logger}
}

func (i TarImageSet) Export(bundleUnprocessedImageRef *UnprocessedImageRef, foundImages *UnprocessedImageRefs, outputPath string, registry ImagesReaderWriter, imageLayerWriterCheck imagetar.ImageLayerWriterFilter) (*imagedesc.ImageRefDescriptors, error) {
	ids, err := i.imageSet.Export(foundImages, registry)
	if err != nil {
		return nil, err
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("Creating file '%s': %s", outputPath, err)
	}

	err = outputFile.Close()
	if err != nil {
		return nil, err
	}

	outputFileOpener := func() (io.WriteCloser, error) {
		return os.OpenFile(outputPath, os.O_RDWR, 0755)
	}

	i.logger.WriteStr("writing layers...\n")

	opts := imagetar.TarWriterOpts{Concurrency: i.concurrency}
	if bundleUnprocessedImageRef != nil {
		for i, descriptor := range ids.Descriptors() {
			if descriptor.Image != nil {
				for _, ref := range descriptor.Image.Refs {
					if ref == bundleUnprocessedImageRef.DigestRef {
						ids.Descriptors()[i].Image.Labels = map[string]interface{}{"main.bundle": "true"}
					}
				}
			}
		}
	}

	return ids, imagetar.NewTarWriter(ids, outputFileOpener, opts, i.logger, imageLayerWriterCheck).Write()
}

func (i *TarImageSet) Import(path string,
	importRepo regname.Repository, registry ImagesReaderWriter) (*ProcessedImage, *ProcessedImages, error) {

	bundleUnprocessedImageRef, imgOrIndexes, err := imagetar.NewTarReader(path).Read()
	if err != nil {
		return nil, nil, err
	}

	processedImages, err := i.imageSet.Import(imgOrIndexes, importRepo, registry)
	if err != nil {
		return nil, nil, err
	}

	var bundleProcessedImageRef *ProcessedImage
	if bundleUnprocessedImageRef != nil {
		processedImg, ok := processedImages.FindByURL(UnprocessedImageRef{
			DigestRef: bundleUnprocessedImageRef.Refs[0],
			Tag:       bundleUnprocessedImageRef.Tag,
		})
		if !ok {
			panic(fmt.Errorf("Internal inconsistency: Unable to find bundle after processing"))
		}
		bundleProcessedImageRef = &processedImg
	}

	return bundleProcessedImageRef, processedImages, err
}
