// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"os"

	regname "github.com/google/go-containerregistry/pkg/name"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
)

type TarImageSet struct {
	imageSet    ImageSet
	concurrency int
	logger      *ctlimg.LoggerPrefixWriter
}

func (o TarImageSet) Export(foundImages *UnprocessedImageURLs,
	outputPath string, registry ctlimg.Registry) error {

	ids, err := o.imageSet.Export(foundImages, registry)
	if err != nil {
		return err
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("Creating file '%s': %s", outputPath, err)
	}

	err = outputFile.Close()
	if err != nil {
		return err
	}

	outputFileOpener := func() (io.WriteCloser, error) {
		return os.OpenFile(outputPath, os.O_RDWR, 0755)
	}

	o.logger.WriteStr("writing layers...\n")

	opts := imagetar.TarWriterOpts{Concurrency: o.concurrency}

	return imagetar.NewTarWriter(ids, outputFileOpener, opts, o.logger).Write()
}

func (o *TarImageSet) Import(path string,
	importRepo regname.Repository, registry ctlimg.Registry) (*ProcessedImages, string, error) {
	//return img or indexes and call image set import later
	imgOrIndexes, err := imagetar.NewTarReader(path).Read()
	if err != nil {
		return nil, "", err
	}

	bundleRef := ""
	for _, imgOrIndex := range imgOrIndexes {
		if imgOrIndex.Index != nil {
			continue
		}

		hasBundle, err := isBundle(*imgOrIndex.Image)
		if err != nil {
			return nil, "", err
		}
		if hasBundle {
			bundleRef = (*imgOrIndex.Image).Ref()
		}
	}
	processedImages, err := o.imageSet.Import(imgOrIndexes, importRepo, registry)
	return processedImages, bundleRef, err
}
