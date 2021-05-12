// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"strings"

	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
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

		foundImg, err := o.imgRetriever.FirstImageExists([]string{imageInBundleRepo, imgRef.Image})
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

func (o *ImagesLock) imageRelativeToBundle(img string) (string, error) {
	imgParts := strings.Split(img, "@")
	if len(imgParts) != 2 {
		panic(fmt.Sprintf("Internal inconsistency: The provided image URL '%s' does not contain a digest", img))
	}
	return o.relativeToRepo + "@" + imgParts[1], nil
}
