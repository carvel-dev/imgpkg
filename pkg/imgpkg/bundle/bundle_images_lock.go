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

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
)

// ImagesRefs Retrieve the references for the Images of this particular bundle
func (o *Bundle) ImagesRefs() []ImageRef {
	return o.cachedImageRefs.All()
}

// AllImagesLockRefs returns a flat list of nested bundles and every image reference for a specific bundle
func (o *Bundle) AllImagesLockRefs(concurrency int, logger util.LoggerWithLevels) ([]*Bundle, ImageRefs, error) {
	throttleReq := util.NewThrottle(concurrency)

	return o.buildAllImagesLock(&throttleReq, logger)
}

// UpdateImageRefs updates the bundle cached images without talking to the registry
func (o *Bundle) UpdateImageRefs(bundles []*Bundle) error {
	o.cachedImageRefs = newImageRefCache()

	img, err := o.checkedImage()
	if err != nil {
		return err
	}

	// Call fetchImagesRef with a NotFoundLocationsConfig because this function should only be used
	// in the copy from tar to repository
	imageRefsToProcess, err := o.fetchImagesRef(img, &NotFoundLocationsConfig{})
	if err != nil {
		return fmt.Errorf("Fetching images of %s: %s", o.DigestRef(), err)
	}

	for _, image := range imageRefsToProcess.ImageRefs() {
		isBundle := false
		for _, bundle := range bundles {
			if bundle.Digest() == image.Digest() {
				isBundle = true
				image.ImageType = BundleImage
				break
			}
		}
		image.IsBundle = &isBundle
		// We want to keep the annotations, only ensure the rest of the information is copied
		o.updateCachedImageRefWithoutAnnotations(image)
	}
	return nil
}

func (o *Bundle) buildAllImagesLock(throttleReq *util.Throttle, logger util.LoggerWithLevels) ([]*Bundle, ImageRefs, error) {
	o.cachedImageRefs = newImageRefCache()

	img, err := o.checkedImage()
	if err != nil {
		return nil, ImageRefs{}, err
	}

	bundleDigestRef, err := regname.NewDigest(o.DigestRef())
	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency: The Bundle Reference '%s' does not have a digest", o.DigestRef()))
	}

	locationsConfig := LocationsConfig{
		logger:          logger,
		imgRetriever:    o.imgRetriever,
		bundleDigestRef: bundleDigestRef,
	}
	imageRefsToProcess, err := o.fetchImagesRef(img, &locationsConfig)
	if err != nil {
		return nil, ImageRefs{}, err
	}

	processedImageRefs := NewImageRefs()
	bundles := []*Bundle{o}

	errChan := make(chan error, len(imageRefsToProcess.ImageRefs()))
	mutex := &sync.Mutex{}

	for _, image := range imageRefsToProcess.ImageRefs() {
		o.cachedImageRefs.StoreImageRef(image.DeepCopy())

		// Check if this image is not a bundle and skips
		if image.IsBundle != nil && *image.IsBundle == false {
			typedImageRef := NewContentImageRef(image.ImageRef).DeepCopy()
			processedImageRefs.AddImagesRef(typedImageRef)
			o.cachedImageRefs.StoreImageRef(typedImageRef)
			errChan <- nil
			continue
		}

		image := image.DeepCopy()
		go func() {
			nestedBundles, nestedBundlesProcessedImageRefs, imgRef, err := o.imagesLockIfIsBundle(throttleReq, image, logger)
			if err != nil {
				errChan <- err
				return
			}

			mutex.Lock()
			defer mutex.Unlock()
			bundles = append(bundles, nestedBundles...)

			// Adds Image to the resulting ImagesLock
			isBundle := len(nestedBundles) > 0 // nestedBundles have Bundles when the image is a bundle
			var typedImgRef ImageRef
			if isBundle {
				typedImgRef = NewBundleImageRef(imgRef)
			} else {
				typedImgRef = NewContentImageRef(imgRef)
			}
			o.cachedImageRefs.StoreImageRef(typedImgRef)
			processedImageRefs.AddImagesRef(typedImgRef)

			processedImageRefs.AddImagesRef(nestedBundlesProcessedImageRefs.ImageRefs()...)
			errChan <- nil
		}()
	}

	for range imageRefsToProcess.ImageRefs() {
		if err := <-errChan; err != nil {
			return nil, ImageRefs{}, err
		}
	}

	return bundles, processedImageRefs, nil
}

func (o *Bundle) fetchImagesRef(img regv1.Image, locationsConfig ImageRefLocationsConfig) (ImageRefs, error) {
	// Reads the ImagesLock of the bundle because this is the source of truth
	imagesLock, err := o.imagesLockReader.Read(img)
	if err != nil {
		return ImageRefs{}, fmt.Errorf("Reading ImagesLock file: %s", err)
	}

	// We use ImagesLock struct only to add the bundle repository to the list of locations
	// maybe we can move this functionality to the bundle in the future
	refs, err := NewImageRefsFromImagesLock(imagesLock, locationsConfig)
	if err != nil {
		return ImageRefs{}, err
	}

	refs.LocalizeToRepo(o.Repo())

	return refs, nil
}

func (o *Bundle) imagesLockIfIsBundle(throttleReq *util.Throttle, imgRef ImageRef, logger util.LoggerWithLevels) ([]*Bundle, ImageRefs, lockconfig.ImageRef, error) {
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

	bundle := NewBundle(newImgRef.PrimaryLocation(), o.imgRetriever, o.imagesLockReader)

	throttleReq.Take()
	isBundle, err := bundle.IsBundle()
	throttleReq.Done()
	if err != nil {
		return nil, ImageRefs{}, lockconfig.ImageRef{}, fmt.Errorf("Checking if '%s' is a bundle: %s", imgRef.Image, err)
	}

	var processedImageRefs ImageRefs
	var nestedBundles []*Bundle
	if isBundle {
		nestedBundles, processedImageRefs, err = bundle.buildAllImagesLock(throttleReq, logger)
		if err != nil {
			return nil, ImageRefs{}, lockconfig.ImageRef{}, fmt.Errorf("Retrieving images for bundle '%s': %s", imgRef.Image, err)
		}
	}
	return nestedBundles, processedImageRefs, newImgRef, nil
}

func NewImagesLockReader() *SingleLayerReader {
	return &SingleLayerReader{
		imagesLock:      map[string]lockconfig.ImagesLock{},
		imagesLockMutex: &sync.Mutex{},
	}
}

type SingleLayerReader struct {
	imagesLock      map[string]lockconfig.ImagesLock
	imagesLockMutex *sync.Mutex
}

func (o *SingleLayerReader) Read(img regv1.Image) (lockconfig.ImagesLock, error) {
	imagesLock, found := o.cachedImagesLock(img)
	if found {
		return imagesLock, nil
	}
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

	imgLock, err := lockconfig.NewImagesLockFromBytes(bs)
	if err != nil {
		digest, dErr := img.Digest()
		if dErr != nil {
			panic(fmt.Sprintf("Internal inconsistency: unable to retrieve digest for image with error: '%s', also with unmarshalling error: %s", dErr, err))
		}
		return conf, fmt.Errorf("Unmarshalling ImagesLock from image with Digest '%s': %s", digest, err)
	}
	o.storeImagesLock(img, imgLock)
	return imgLock, nil
}

func (o *SingleLayerReader) cachedImagesLock(img regv1.Image) (lockconfig.ImagesLock, bool) {
	digestHash, err := img.Digest()
	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency, unable to get Digest: %s", err))
	}
	o.imagesLockMutex.Lock()
	defer o.imagesLockMutex.Unlock()

	imgsLock, found := o.imagesLock[digestHash.String()]
	return imgsLock, found
}

func (o *SingleLayerReader) storeImagesLock(img regv1.Image, lock lockconfig.ImagesLock) {
	digestHash, err := img.Digest()
	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency, unable to get Digest: %s", err))
	}
	o.imagesLockMutex.Lock()
	defer o.imagesLockMutex.Unlock()

	o.imagesLock[digestHash.String()] = lock
}

type LocationsConfig struct {
	logger          util.LoggerWithLevels
	imgRetriever    ImagesMetadata
	bundleDigestRef regname.Digest
}

func (l LocationsConfig) Config() (ImageLocationsConfig, error) {
	return NewLocations(l.logger).Fetch(l.imgRetriever, l.bundleDigestRef)
}

// NotFoundLocationsConfig Noop Locations Configuration retrieval
type NotFoundLocationsConfig struct{}

// Config Returns a LocationsNotFound error
func (l NotFoundLocationsConfig) Config() (ImageLocationsConfig, error) {
	return ImageLocationsConfig{}, &LocationsNotFound{}
}
