// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imageset

import (
	"fmt"
	"io"
	"os"

	goui "github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
)

type TarImageSet struct {
	imageSet    ImageSet
	concurrency int
	ui          goui.UI
}

// NewTarImageSet provides export/import operations on a tarball for a set of images
func NewTarImageSet(imageSet ImageSet, concurrency int, ui goui.UI) TarImageSet {
	return TarImageSet{imageSet, concurrency, ui}
}

func (i TarImageSet) Export(foundImages *UnprocessedImageRefs, preloadedImages *ProcessedImages, outputPath string, registry ImagesReaderWriter, imageLayerWriterCheck imagetar.ImageLayerWriterFilter) (*imagedesc.ImageRefDescriptors, error) {
	ids, err := i.imageSet.Export(foundImages, preloadedImages, registry)
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

	i.ui.BeginLinef("writing layers...\n")

	opts := imagetar.TarWriterOpts{Concurrency: i.concurrency}

	return ids, imagetar.NewTarWriter(ids, outputFileOpener, opts, i.ui, imageLayerWriterCheck).Write()
}

func (i *TarImageSet) Import(path string, importRepo regname.Repository, registry ImagesReaderWriter) (*ProcessedImages, error) {
	imgOrIndexes, err := imagetar.NewTarReader(path).Read()
	if err != nil {
		return nil, err
	}

	processedImages, err := i.imageSet.Import(imgOrIndexes, importRepo, registry)
	if err != nil {
		return nil, err
	}

	return processedImages, err
}
