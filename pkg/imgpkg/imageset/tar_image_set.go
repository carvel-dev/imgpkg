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

const rootBundleLabelKey string = "root.bundle"

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
		err := ids.SetLabel(map[string]interface{}{rootBundleLabelKey: ""}, bundleUnprocessedImageRef.DigestRef)
		if err != nil {
			return nil, err
		}
	}

	return ids, imagetar.NewTarWriter(ids, outputFileOpener, opts, i.logger, imageLayerWriterCheck).Write()
}

func (i *TarImageSet) Import(path string,
	importRepo regname.Repository, registry ImagesReaderWriter) (*ProcessedImage, *ProcessedImages, error) {

	reader := imagetar.NewTarReader(path)
	imgOrIndexes, err := reader.Read()
	if err != nil {
		return nil, nil, err
	}

	rootBundleImageDesc, err := reader.FindByLabelKey(rootBundleLabelKey)
	if err != nil {
		return nil, nil, err
	}

	if len(rootBundleImageDesc) > 1 {
		panic(fmt.Sprintf("Internal inconsistency: The tarball contains multiple images marked as %s", rootBundleLabelKey))
	}

	processedImages, err := i.imageSet.Import(imgOrIndexes, importRepo, registry)
	if err != nil {
		return nil, nil, err
	}

	var bundleProcessedImageRef *ProcessedImage
	if len(rootBundleImageDesc) == 1 {
		processedImg, ok := processedImages.FindByURL(UnprocessedImageRef{
			DigestRef: rootBundleImageDesc[0].Refs[0],
			Tag:       rootBundleImageDesc[0].Tag,
		})
		if !ok {
			panic(fmt.Errorf("Internal inconsistency: Unable to find bundle after processing"))
		}
		bundleProcessedImageRef = &processedImg
	}

	return bundleProcessedImageRef, processedImages, err
}
