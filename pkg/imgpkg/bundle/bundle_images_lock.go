// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sync"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
)

func (o *Bundle) AllImagesLock(concurrency int) (*ImagesLock, error) {
	throttleReq := util.NewThrottle(concurrency)
	return o.buildAllImagesLock(&throttleReq, &processedImages{processedImgs: map[string]struct{}{}})
}

func (o *Bundle) buildAllImagesLock(throttleReq *util.Throttle, processedImgs *processedImages) (*ImagesLock, error) {
	img, err := o.checkedImage()
	if err != nil {
		return nil, err
	}

	imagesLock, err := o.imagesLockReader.Read(img)
	if err != nil {
		return nil, err
	}

	allImagesLock := NewImagesLock(imagesLock, o.imgRetriever, o.Repo())

	errChan := make(chan error, len(imagesLock.Images))
	mutex := &sync.Mutex{}

	for _, image := range imagesLock.Images {
		if skip := processedImgs.CheckAndAddImage(image.Image); skip {
			errChan <- nil
			continue
		}

		image := image.DeepCopy()
		go func() {
			imgsLock, err := o.imagesLockIfIsBundle(throttleReq, image, processedImgs)
			if err != nil {
				errChan <- err
				return
			}
			if imgsLock != nil {
				mutex.Lock()
				defer mutex.Unlock()
				err = allImagesLock.Merge(imgsLock)
				if err != nil {
					errChan <- fmt.Errorf("Merging images for bundle '%s': %s", image.Image, err)
					return
				}
			}
			errChan <- nil
		}()
	}

	for range imagesLock.Images {
		if err := <-errChan; err != nil {
			return nil, err
		}
	}

	err = allImagesLock.GenerateImagesLocations()
	if err != nil {
		return nil, fmt.Errorf("Generating locations list for images in bundle %s: %s", o.DigestRef(), err)
	}

	return allImagesLock, nil
}

func (o *Bundle) imagesLockIfIsBundle(throttleReq *util.Throttle, image lockconfig.ImageRef, processedImgs *processedImages) (*ImagesLock, error) {
	throttleReq.Take()
	bundle := NewBundleWithReader(image.Image, o.imgRetriever, o.imagesLockReader)

	isBundle, err := bundle.IsBundle()
	throttleReq.Done()
	if err != nil {
		return nil, fmt.Errorf("Checking if '%s' is a bundle: %s", image.Image, err)
	}

	var imgLock *ImagesLock
	if isBundle {
		imgLock, err = bundle.buildAllImagesLock(throttleReq, processedImgs)
		if err != nil {
			return nil, fmt.Errorf("Retrieving images for bundle '%s': %s", image.Image, err)
		}
	}
	return imgLock, nil
}

type processedImages struct {
	lock          sync.Mutex
	processedImgs map[string]struct{}
}

func (p *processedImages) CheckAndAddImage(ref string) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	_, present := p.processedImgs[ref]
	p.processedImgs[ref] = struct{}{}
	return present
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
