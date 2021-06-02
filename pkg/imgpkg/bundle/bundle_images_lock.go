// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
)

func (o *Bundle) AllImagesRefs(concurrency int, logger util.LoggerWithLevels) ([]*Bundle, ImageRefs, error) {
	throttleReq := util.NewThrottle(concurrency)
	bundles, imageRefs, err := o.buildAllImagesLock(&throttleReq, &processedImages{processedImgs: map[string]struct{}{}}, logger)
	if err != nil {
		return nil, ImageRefs{}, err
	}

	// Ensure that the correct IsBundle flag is provided.
	// This loop needs to happen because we skipped some images for some bundle, and only at this point we have
	// the full list of ImageRefs created and can fill the gaps inside each bundle
	for _, bundle := range bundles {
		for _, ref := range bundle.ImageRefs() {
			imgRef, found := imageRefs.Find(ref.Image)
			if !found {
				panic(fmt.Sprintf("Internal inconsistency: The Image '%s' cannot be found in the total list of images", ref.Image))
			}

			bundle.AddImageRefs(imgRef)
		}
	}

	return bundles, imageRefs, err
}

func (o *Bundle) buildAllImagesLock(throttleReq *util.Throttle, processedImgs *processedImages, logger util.LoggerWithLevels) ([]*Bundle, ImageRefs, error) {
	img, err := o.checkedImage()
	if err != nil {
		return nil, ImageRefs{}, err
	}

	imageRefsToProcess, allImageRefs, err := o.fetchImagesRef(img, logger)
	if err != nil {
		return nil, ImageRefs{}, err
	}

	bundles := []*Bundle{o}

	errChan := make(chan error, len(imageRefsToProcess.ImageRefs()))
	mutex := &sync.Mutex{}

	for _, image := range imageRefsToProcess.ImageRefs() {
		o.AddImageRefs(image)

		if skip := processedImgs.CheckAndAddImage(image.Image); skip {
			errChan <- nil
			continue
		}

		image := image.DeepCopy()
		go func() {
			nestedBundles, nestedBundlesProcessedImageRefs, imgRef, err := o.imagesLockIfIsBundle(throttleReq, image, processedImgs, logger)
			if err != nil {
				errChan <- err
				return
			}

			mutex.Lock()
			defer mutex.Unlock()
			if nestedBundles != nil {
				bundles = append(bundles, nestedBundles...)
			}

			// Adds Image to the resulting ImagesLock
			allImageRefs.AddImagesRef(ImageRef{
				ImageRef: imgRef,
				IsBundle: nestedBundles != nil, // nestedBundles will be != nil when the image is a bundle
			})
			allImageRefs.AddImagesRef(nestedBundlesProcessedImageRefs.ImageRefs()...)
			errChan <- nil
		}()
	}

	for range imageRefsToProcess.ImageRefs() {
		if err := <-errChan; err != nil {
			return nil, ImageRefs{}, err
		}
	}

	return bundles, allImageRefs, nil
}

func (o *Bundle) fetchImagesRef(img regv1.Image, logger util.LoggerWithLevels) (ImageRefs, ImageRefs, error) {
	bundleDigestRef, err := regname.NewDigest(o.DigestRef())
	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency: The Bundle Reference '%s' does not have a digest", o.DigestRef()))
	}

	// Reads the ImagesLock of the bundle because this is the source of truth
	imagesLock, err := o.imagesLockReader.Read(img)
	if err != nil {
		return ImageRefs{}, ImageRefs{}, err
	}

	// We use ImagesLock struct only to add the bundle repository to the list of locations
	// maybe we can move this functionality to the bundle in the future
	currentImagesLock := NewImagesLock(imagesLock, o.imgRetriever, o.Repo())
	imageRefsToProcess := currentImagesLock.ImageRefs()
	processedImageRefs := ImageRefs{}

	locationsConfig, err := NewLocations(logger).Fetch(o.imgRetriever, bundleDigestRef)
	if err == nil {
		imageRefsToProcess, processedImageRefs = o.processLocations(imageRefsToProcess, locationsConfig)
	} else if _, ok := err.(*LocationsNotFound); !ok {
		return ImageRefs{}, ImageRefs{}, err
	}
	return imageRefsToProcess, processedImageRefs, nil
}

func (o *Bundle) processLocations(imageRefs ImageRefs, locationsConfig ImageLocationsConfig) (ImageRefs, ImageRefs) {
	unprocessedImageRefes := imageRefs.DeepCopy()
	processedImageRefs := ImageRefs{}
	for _, imgRef := range imageRefs.ImageRefs() {
		for _, image := range locationsConfig.Images {
			if image.Image == imgRef.Image {
				// We need to keep all the ImagesLock information and the only added pieces are the new location and
				// if this image is a bundle or not
				imgRef := imgRef.DeepCopy()
				imgRef.IsBundle = image.IsBundle

				imgParts := strings.Split(image.Image, "@")
				if len(imgParts) != 2 {
					panic(fmt.Sprintf("Internal inconsistency: The provided image URL '%s' does not contain a digest", image.Image))
				}
				imgRef.AddLocation(o.Repo() + "@" + imgParts[1])

				if !image.IsBundle {
					// When the image is not a bundle we do not need to process it again.
					o.AddImageRefs(imgRef)
				} else {
					unprocessedImageRefes.AddImagesRef(imgRef)
				}
			}
		}
	}

	return unprocessedImageRefes, processedImageRefs
}

func (o *Bundle) imagesLockIfIsBundle(throttleReq *util.Throttle, imgRef ImageRef, processedImgs *processedImages, levels util.LoggerWithLevels) ([]*Bundle, ImageRefs, lockconfig.ImageRef, error) {
	throttleReq.Take()
	// We need to check where we can find the image we are looking for.
	// First checks the current bundle repository and if it cannot be found there
	// it will check in the original location of the image
	imgURL, err := o.imgRetriever.FirstImageExists(imgRef.Locations())
	throttleReq.Done()
	if err != nil {
		return nil, ImageRefs{}, lockconfig.ImageRef{}, err
	}
	newImgRef := imgRef.DiscardLocationsExcept(imgURL)

	bundle := NewBundleWithReader(newImgRef.PrimaryLocation(), o.imgRetriever, o.imagesLockReader)

	throttleReq.Take()
	isBundle, err := bundle.IsBundle()
	throttleReq.Done()
	if err != nil {
		return nil, ImageRefs{}, lockconfig.ImageRef{}, fmt.Errorf("Checking if '%s' is a bundle: %s", imgRef.Image, err)
	}

	var processedImageRefs ImageRefs
	var nestedBundles []*Bundle
	if isBundle {
		nestedBundles, processedImageRefs, err = bundle.buildAllImagesLock(throttleReq, processedImgs, levels)
		if err != nil {
			return nil, ImageRefs{}, lockconfig.ImageRef{}, fmt.Errorf("Retrieving images for bundle '%s': %s", imgRef.Image, err)
		}
	}
	return nestedBundles, processedImageRefs, newImgRef, nil
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
