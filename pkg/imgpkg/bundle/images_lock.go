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
	IsBundle *bool
}

func (i ImageRef) DeepCopy() ImageRef {
	return ImageRef{
		ImageRef: i.ImageRef.DeepCopy(),
		IsBundle: i.IsBundle,
	}
}
func NewImageRef(imgRef lockconfig.ImageRef, isBundle bool) ImageRef {
	return ImageRef{ImageRef: imgRef, IsBundle: &isBundle}
}

type ImageRefs struct {
	refs []ImageRef
}

func (i *ImageRefs) AddImagesRef(refs ...ImageRef) {
	for _, ref := range refs {
		found := false
		for j, imageRef := range i.refs {
			if imageRef.Image == ref.Image {
				found = true
				i.refs[j] = ref.DeepCopy()
				break
			}
		}

		if !found {
			i.refs = append(i.refs, ref)
		}
	}
}
func (i *ImageRefs) Find(ref string) (ImageRef, bool) {
	for _, imageRef := range i.refs {
		if imageRef.Image == ref {
			return imageRef, true
		}
	}

	return ImageRef{}, false
}
func (i ImageRefs) ImageRefs() []ImageRef {
	return i.refs
}
func (i ImageRefs) DeepCopy() ImageRefs {
	var result ImageRefs
	result.AddImagesRef(i.ImageRefs()...)
	return result
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesLockLocationConfig
type ImagesLockLocationConfig interface {
	Fetch() (ImageLocationsConfig, error)
}

func NewImagesLock(imagesLock lockconfig.ImagesLock, imgRetriever ctlimg.ImagesMetadata, relativeToRepo string, imagesLockLocationConfig ImagesLockLocationConfig) *ImagesLock {
	imageRefs := ImageRefs{}
	for _, image := range imagesLock.Images {
		imageRefs.AddImagesRef(ImageRef{ImageRef: image, IsBundle: nil})
	}

	imgsLock := &ImagesLock{imageRefs: imageRefs, imgRetriever: imgRetriever, imagesLockLocationFetcher: imagesLockLocationConfig}
	imgsLock.generateImagesLocations(relativeToRepo)

	return imgsLock
}

func (o *ImagesLock) SyncImageRefs() error {
	locationsConfig, err := o.imagesLockLocationFetcher.Fetch()
	if _, ok := err.(*LocationsNotFound); ok {
		return nil
	}

	if err != nil {
		return err
	}

	for _, imgRef := range o.imageRefs.ImageRefs() {
		for _, imgLoc := range locationsConfig.Images {
			if imgLoc.Image == imgRef.Image {
				isBundle := imgLoc.IsBundle
				imgRef.IsBundle = &isBundle
			}
		}
	}

	return nil
}

type ImagesLock struct {
	imageRefs                 ImageRefs
	imgRetriever              ctlimg.ImagesMetadata
	imagesLockLocationFetcher ImagesLockLocationConfig
}

func (o *ImagesLock) generateImagesLocations(relativeToRepo string) {
	result := ImageRefs{}
	for _, imgRef := range o.imageRefs.ImageRefs() {
		imageInBundleRepo := o.imageRelativeToBundle(imgRef.Image, relativeToRepo)
		imgRef.AddLocation(imageInBundleRepo)

		result.AddImagesRef(imgRef)
	}
	o.imageRefs = result
}

func (o ImagesLock) ImageRefs() ImageRefs {
	return o.imageRefs
}
func (o *ImagesLock) Merge(imgLock *ImagesLock) {
	o.imageRefs.AddImagesRef(imgLock.imageRefs.ImageRefs()...)
}

func (o *ImagesLock) AddImageRef(ref lockconfig.ImageRef, bundle bool) {
	o.imageRefs.AddImagesRef(NewImageRef(ref.DeepCopy(), bundle))
}

func (o *ImagesLock) LocalizeImagesLock() (lockconfig.ImagesLock, bool, error) {
	var imageRefs []lockconfig.ImageRef
	imagesLock := lockconfig.NewEmptyImagesLock()

	_, err := o.imagesLockLocationFetcher.Fetch()
	// location fetcher was unable to get the Location OCI. Either it is missing or there was an error fetching it
	// Either way, revert back to checking if each image has been relocated to determine whether bundle should be Localized
	if err != nil {
		skippedLocalization := false

		for _, imgRef := range o.imageRefs.ImageRefs() {
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
			for _, image := range o.imageRefs.ImageRefs() {
				imageRefs = append(imageRefs, image.DiscardLocationsExcept(image.Image))
			}
		}

		imagesLock.Images = imageRefs
		return imagesLock, skippedLocalization, nil
	}

	for _, imgRef := range o.imageRefs.ImageRefs() {
		imageRefs = append(imageRefs, lockconfig.ImageRef{
			Image:       imgRef.PrimaryLocation(),
			Annotations: imgRef.Annotations,
		})
	}

	imagesLock.Images = imageRefs
	return imagesLock, false, nil
}

func (o *ImagesLock) imageRelativeToBundle(img, relativeToRepo string) string {
	imgParts := strings.Split(img, "@")
	if len(imgParts) != 2 {
		panic(fmt.Sprintf("Internal inconsistency: The provided image URL '%s' does not contain a digest", img))
	}
	return relativeToRepo + "@" + imgParts[1]
}
