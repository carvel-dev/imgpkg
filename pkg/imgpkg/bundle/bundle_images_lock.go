// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

func (o *Bundle) AllImagesLock() (*ImagesLock, error) {
	return o.buildAllImagesLock(map[string]struct{}{})
}

func (o *Bundle) buildAllImagesLock(processedImgs map[string]struct{}) (*ImagesLock, error) {
	img, err := o.checkedImage()
	if err != nil {
		return nil, err
	}

	imagesLock, err := o.imagesLockReader.Read(img)
	if err != nil {
		return nil, err
	}

	allImagesLock := NewImagesLock(imagesLock, o.imgRetriever, o.Repo())

	for _, image := range imagesLock.Images {
		if _, skip := processedImgs[image.Image]; skip {
			continue
		}
		processedImgs[image.Image] = struct{}{}

		bundle := NewBundleWithReader(image.Image, o.imgRetriever, o.imagesLockReader)
		isBundle, err := bundle.IsBundle()
		if err != nil {
			return nil, fmt.Errorf("Checking if '%s' is a bundle: %s", image.Image, err)
		}

		if isBundle {
			imgLock, err := bundle.buildAllImagesLock(processedImgs)
			if err != nil {
				return nil, fmt.Errorf("Retrieving images for bundle '%s': %s", image.Image, err)
			}

			err = allImagesLock.Merge(imgLock)
			if err != nil {
				return nil, fmt.Errorf("Merging images for bundle '%s': %s", image.Image, err)
			}
		}
	}

	err = allImagesLock.GenerateImagesLocations()
	if err != nil {
		return nil, fmt.Errorf("Generating locations list for images in bundle %s: %s", o.DigestRef(), err)
	}

	return allImagesLock, nil
}

// ImagesLockLocalized returns possibly modified images lock
// with image URLs relative to bundle location
func (o *Bundle) ImagesLockLocalized() (lockconfig.ImagesLock, error) {
	img, err := o.checkedImage()
	if err != nil {
		return lockconfig.ImagesLock{}, err
	}

	imagesLock, err := o.imagesLockReader.Read(img)
	if err != nil {
		return lockconfig.ImagesLock{}, err
	}

	imagesLock, _, err = NewImagesLock(imagesLock, o.imgRetriever, o.Repo()).LocalizeImagesLock(true)
	return imagesLock, err
}

type singleLayerReader struct{}

func (o *singleLayerReader) Read(img regv1.Image) (lockconfig.ImagesLock, error) {
	conf := lockconfig.ImagesLock{}

	layers, err := img.Layers()
	if err != nil {
		return conf, err
	}

	if len(layers) != 1 {
		return conf, fmt.Errorf("Expected bundle to only have a single layer, got %d", len(layers))
	}

	layer := layers[0]

	mediaType, err := layer.MediaType()
	if err != nil {
		return conf, err
	}

	if mediaType != types.DockerLayer {
		return conf, fmt.Errorf("Expected layer to have docker layer media type, was %s", mediaType)
	}

	// here we know layer is .tgz so decompress and read tar headers
	unzippedReader, err := layer.Uncompressed()
	if err != nil {
		return conf, fmt.Errorf("Could not read bundle image layer contents: %v", err)
	}

	tarReader := tar.NewReader(unzippedReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				return conf, fmt.Errorf("Expected to find .imgpkg/images.yml in bundle image")
			}
			return conf, fmt.Errorf("reading tar: %v", err)
		}

		basename := filepath.Base(header.Name)
		dirname := filepath.Dir(header.Name)
		if dirname == ImgpkgDir && basename == ImagesLockFile {
			break
		}
	}

	bs, err := ioutil.ReadAll(tarReader)
	if err != nil {
		return conf, fmt.Errorf("Reading images.yml from layer: %s", err)
	}

	return lockconfig.NewImagesLockFromBytes(bs)
}
