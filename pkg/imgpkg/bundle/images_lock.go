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

type ImageRef struct {
	lockconfig.ImageRef
	IsBundle bool
}

func NewImagesLock(imagesLock lockconfig.ImagesLock, imgRetriever ctlimg.ImagesMetadata, relativeToRepo string) *ImagesLock {
	var imagesRef []ImageRef
	for _, image := range imagesLock.Images {
		imagesRef = append(imagesRef, ImageRef{ImageRef: image, IsBundle: false})
	}

	imgsLock := &ImagesLock{imagesRef: imagesRef, imgRetriever: imgRetriever}
	imgsLock.generateImagesLocations(relativeToRepo)
	return imgsLock
}

type ImagesLock struct {
	imagesRef    []ImageRef
	imgRetriever ctlimg.ImagesMetadata
}

func (o *ImagesLock) generateImagesLocations(relativeToRepo string) {
	for i, imgRef := range o.imagesRef {
		imageInBundleRepo := o.imageRelativeToBundle(imgRef.Image, relativeToRepo)
		o.imagesRef[i].AddLocation(imageInBundleRepo)
	}
}

func (o ImagesLock) ImageRefs() []ImageRef {
	return o.imagesRef
}

func (o *ImagesLock) Merge(imgLock *ImagesLock) error {
	for _, image := range imgLock.imagesRef {
		imgRef := image.DeepCopy()
		o.AddImageRef(imgRef, image.IsBundle)
	}

	return nil
}

func (o *ImagesLock) AddImageRef(ref lockconfig.ImageRef, bundle bool) {
	for _, image := range o.imagesRef {
		if image.Image == ref.Image {
			return
		}
	}
	o.imagesRef = append(o.imagesRef, ImageRef{ImageRef: ref.DeepCopy(), IsBundle: bundle})
}

func (o *ImagesLock) LocalizeImagesLock() (lockconfig.ImagesLock, bool, error) {
	var imageRefs []lockconfig.ImageRef
	imagesLock := lockconfig.NewEmptyImagesLock()

	skippedLocalization := false
	for _, imgRef := range o.imagesRef {
		foundImg, err := o.imgRetriever.FirstImageExists(imgRef.Locations())
		if err != nil {
			return lockconfig.ImagesLock{}, false, err
		}

		// If cannot find the image in the bundle repo, will not localize any image
		// We assume that the bundle was not copied to the bundle location,
		// so there we cannot localize any image
		if foundImg != imgRef.PrimaryLocation() {
			skippedLocalization = true
			break
		}

		imageRefs = append(imageRefs, lockconfig.ImageRef{
			Image:       foundImg,
			Annotations: imgRef.Annotations,
		})
	}

	if skippedLocalization {
		imageRefs = []lockconfig.ImageRef{}
		// Remove the bundle location on all the Images, which is present due to the constructor call to
		// ImagesLock.generateImagesLocations
		for _, image := range o.imagesRef {
			imageRefs = append(imageRefs, image.DiscardLocationsExcept(image.Image))
		}
	}

	imagesLock.Images = imageRefs
	return imagesLock, skippedLocalization, nil
}

func (o *ImagesLock) imageRelativeToBundle(img, relativeToRepo string) string {
	imgParts := strings.Split(img, "@")
	if len(imgParts) != 2 {
		panic(fmt.Sprintf("Internal inconsistency: The provided image URL '%s' does not contain a digest", img))
	}
	return relativeToRepo + "@" + imgParts[1]
}
