// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"strings"
	"sync"

	regname "github.com/google/go-containerregistry/pkg/name"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 github.com/cppforlife/go-cli-ui/ui.UI

func NewImagesLock(imagesLock lockconfig.ImagesLock, imgRetriever ctlimg.ImagesMetadata, relativeToRepo string) *ImagesLock {
	return &ImagesLock{imagesLock: imagesLock, imgRetriever: imgRetriever, relativeToRepo: relativeToRepo}
}

type ImagesLock struct {
	imagesLock     lockconfig.ImagesLock
	imgRetriever   ctlimg.ImagesMetadata
	relativeToRepo string
}

func (o ImagesLock) ImageRefs() []lockconfig.ImageRef {
	return o.imagesLock.Images
}

func (o *ImagesLock) Merge(imgLock *ImagesLock) error {
	for _, image := range imgLock.imagesLock.Images {
		imgRef := image.DeepCopy()
		o.imagesLock.AddImageRef(imgRef)
	}

	return nil
}

func (o *ImagesLock) GenerateImagesLocations() error {
	for i, imgRef := range o.ImageRefs() {
		imageInBundleRepo, err := o.imageRelativeToBundle(imgRef.Image)
		if err != nil {
			return err
		}

		o.imagesLock.Images[i].AddLocation(imageInBundleRepo)
	}
	return nil
}

func (o *ImagesLock) AddImageRef(ref lockconfig.ImageRef) {
	o.imagesLock.AddImageRef(ref)
}

// TODO: we should use LocationPrunedImageRefs as part of this function
func (o *ImagesLock) LocalizeImagesLock() (lockconfig.ImagesLock, bool, error) {
	var imageRefs []lockconfig.ImageRef
	imagesLock := lockconfig.ImagesLock{
		LockVersion: o.imagesLock.LockVersion,
	}

	for _, imgRef := range o.imagesLock.Images {
		imageInBundleRepo, err := o.imageRelativeToBundle(imgRef.Image)
		if err != nil {
			return o.imagesLock, false, err
		}

		foundImg, err := o.checkImagesExist([]string{imageInBundleRepo, imgRef.Image})
		if err != nil {
			return o.imagesLock, false, err
		}

		// If cannot find the image in the bundle repo, will not localize any image
		// We assume that the bundle was not copied to the bundle location,
		// so there we cannot localize any image
		if foundImg != imageInBundleRepo {
			return o.imagesLock, true, nil
		}

		imageRefs = append(imageRefs, lockconfig.ImageRef{
			Image:       foundImg,
			Annotations: imgRef.Annotations,
		})
	}

	imagesLock.Images = imageRefs
	return imagesLock, false, nil
}

func (o *ImagesLock) LocationPrunedImageRefs(concurrency int) ([]lockconfig.ImageRef, error) {
	var imageRefs []lockconfig.ImageRef

	errChan := make(chan error, len(o.ImageRefs()))
	throttle := util.NewThrottle(concurrency)
	mutex := &sync.Mutex{}

	for _, imgRef := range o.ImageRefs() {
		newImgRef := imgRef.DeepCopy()
		go func() {
			throttle.Take()
			defer throttle.Done()

			foundImg, err := o.checkImagesExist(newImgRef.Locations())
			if err != nil {
				errChan <- err
				return
			}

			newImgRef.DiscardLocationsExcept(foundImg)

			mutex.Lock()
			defer mutex.Unlock()

			imageRefs = append(imageRefs, newImgRef)
			errChan <- nil
		}()
	}

	for range o.ImageRefs() {
		err := <-errChan
		if err != nil {
			return nil, err
		}
	}

	return imageRefs, nil
}

func (o *ImagesLock) checkImagesExist(urls []string) (string, error) {
	var err error
	for _, img := range urls {
		ref, parseErr := regname.NewDigest(img)
		if parseErr != nil {
			return "", parseErr
		}
		_, err = o.imgRetriever.Digest(ref)
		if err == nil {
			return img, nil
		}
	}
	return "", fmt.Errorf("Checking image existence: %s", err)
}

func (o *ImagesLock) imageRelativeToBundle(img string) (string, error) {
	imgParts := strings.Split(img, "@")
	if len(imgParts) != 2 {
		return "", fmt.Errorf("Parsing image URL: %s", img)
	}
	return o.relativeToRepo + "@" + imgParts[1], nil
}
