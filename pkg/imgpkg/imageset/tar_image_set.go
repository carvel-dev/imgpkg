// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imageset

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

func NewTarImageSet(imageSet ImageSet, concurrency int, logger *ctlimg.LoggerPrefixWriter) TarImageSet {
	return TarImageSet{imageSet, concurrency, logger}
}

func (o TarImageSet) Export(foundImages *UnprocessedImageRefs, outputPath string, registry ctlimg.ImagesMetadata, distributable bool) error {
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

	return imagetar.NewTarWriter(ids, outputFileOpener, opts, o.logger, distributable).Write()
}

func (o *TarImageSet) Import(path string,
	importRepo regname.Repository, registry ctlimg.ImagesReaderWriter) (*ProcessedImages, error) {

	imgOrIndexes, err := imagetar.NewTarReader(path).Read()
	if err != nil {
		return nil, err
	}

	processedImages, err := o.imageSet.Import(imgOrIndexes, importRepo, registry)
	return processedImages, err
}
