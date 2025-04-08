// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package imageset

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"carvel.dev/imgpkg/pkg/imgpkg/imagedesc"
	"carvel.dev/imgpkg/pkg/imgpkg/imagetar"
	"carvel.dev/imgpkg/pkg/imgpkg/registry"
	regname "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type TarImageSet struct {
	imageSet    ImageSet
	concurrency int
	logger      Logger
}

// NewTarImageSet provides export/import operations on a tarball for a set of images
func NewTarImageSet(imageSet ImageSet, concurrency int, logger Logger) TarImageSet {
	return TarImageSet{imageSet, concurrency, logger}
}

// Export Creates a Tar with the provided Images
func (i *TarImageSet) Export(foundImages *UnprocessedImageRefs, outputPath string, registry registry.ImagesReaderWriter, imageLayerWriterCheck imagetar.ImageLayerWriterFilter, resume bool) (d *imagedesc.ImageRefDescriptors, err error) {
	ids, err := i.imageSet.Export(foundImages, registry)
	if err != nil {
		return nil, err
	}

	var outputFile *os.File
	var alreadyDownloadedLayers []v1.Layer

	// this temporary file is used only in the case were we are resuming the copy of an image to a tar
	// we are creating a temporary copy of the existing tar. This is done to be able to read the layers
	// when we are filling up the destination tar.
	var tmpFolder, tmpFilename string
	if resume {
		// If the file cannot be open we assume that this is not a resume action.
		// This will just follow the normal path of resume == false
		outputFile, err = os.Open(outputPath)
		if err == nil {
			err := outputFile.Close()
			if err != nil {
				return nil, err
			}
			tmpFolder, err = os.MkdirTemp("", "imgpkg-tar-imageset-")
			if err != nil {
				return nil, fmt.Errorf("Creating tmp folder: %s", err)
			}
			tmpFilename = filepath.Join(tmpFolder, "imgpkg-tar-imageset.tmp")
			cErr := os.Rename(outputPath, tmpFilename)
			if cErr != nil {
				return nil, fmt.Errorf("Moving tar to temporary location: %s", cErr)
			}

			start := time.Now()
			reader := imagetar.NewTarReader(tmpFilename, i.concurrency)
			alreadyDownloadedLayers, err = reader.PresentLayers()
			if err != nil {
				return nil, fmt.Errorf("Reading previously created tar '%s': %s", outputPath, err)
			}
			i.logger.Debugf("Took %s to find all valid layers in tar", time.Since(start))

			i.logger.Logf("Going to reuse %d layers from the tar already in disk\n", len(alreadyDownloadedLayers))
		}
	}

	outputFile, err = os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("Creating file '%s': %s", outputPath, err)
	}
	err = outputFile.Close()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			if tmpFolder != "" {
				os.RemoveAll(tmpFolder)
			}
			return
		}
		if tmpFilename != "" {
			cErr := os.Rename(tmpFilename, outputPath)
			if cErr != nil {
				err = fmt.Errorf("original error: %s, post exit error: %s", err, cErr)
				return
			}
		}
	}()

	outputFileOpener := func() (io.WriteCloser, error) {
		return os.OpenFile(outputPath, os.O_RDWR, 0755)
	}

	i.logger.Logf("writing layers...\n")

	opts := imagetar.TarWriterOpts{Concurrency: i.concurrency}

	err = imagetar.NewTarWriter(ids, outputFileOpener, opts, i.logger, imageLayerWriterCheck, alreadyDownloadedLayers).Write()
	return ids, err
}

// Import Copy tar with Images to the Registry
func (i *TarImageSet) Import(path string, importRepo regname.Repository, registry registry.ImagesReaderWriter) (*ProcessedImages, error) {
	imgOrIndexes, err := imagetar.NewTarReader(path, i.concurrency).Read()
	if err != nil {
		return nil, err
	}

	processedImages, err := i.imageSet.Import(imgOrIndexes, importRepo, registry)
	if err != nil {
		return nil, err
	}

	return processedImages, err
}
