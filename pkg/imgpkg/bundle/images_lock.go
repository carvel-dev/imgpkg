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
	var isBundle *bool
	if i.IsBundle != nil {
		tmp := *i.IsBundle
		isBundle = &tmp
	}

	return ImageRef{
		ImageRef: i.ImageRef.DeepCopy(),
		IsBundle: isBundle,
	}
}
func (i ImageRef) DiscardLocationsExcept(viableLocation string) ImageRef {
	var isBundle *bool
	if i.IsBundle != nil {
		tmp := *i.IsBundle
		isBundle = &tmp
	}

	return ImageRef{
		ImageRef: i.ImageRef.DiscardLocationsExcept(viableLocation),
		IsBundle: isBundle,
	}
}
func NewImageRef(imgRef lockconfig.ImageRef, isBundle bool) ImageRef {
	return ImageRef{ImageRef: imgRef, IsBundle: &isBundle}
}

type ImageRefs struct {
	refs                       []ImageRef
	imagesCollocatedWithBundle *bool
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesLockLocationConfig
type ImagesLockLocationConfig interface {
	Fetch() (ImageLocationsConfig, error)
}

func NewImageRefsFromLock(imagesLock lockconfig.ImagesLock) ImageRefs {
	imageRefs := ImageRefs{}
	for _, image := range imagesLock.Images {
		imageRefs.AddImagesRef(ImageRef{ImageRef: image, IsBundle: nil})
	}

	return imageRefs
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
	imagesCollocatedWithBundle := *i.imagesCollocatedWithBundle
	result.imagesCollocatedWithBundle = &imagesCollocatedWithBundle
	return result
}
func (i ImageRefs) LocalizeAndFindImages(imgRetriever ctlimg.ImagesMetadata, imagesLockLocationConfig ImagesLockLocationConfig, relativeToRepo string) (ImageRefs, error) {
	imgRefs, err := i.LocalizeToBundle(imagesLockLocationConfig, relativeToRepo)
	if err != nil {
		return ImageRefs{}, err
	}

	newStuff, err := imgRefs.imageRefsFound(imgRetriever)
	if err != nil {
		return ImageRefs{}, err
	}

	imagesCollocatedWithBundle := newStuff.CollocatedWithBundle()
	if !imagesCollocatedWithBundle {
		i.imagesCollocatedWithBundle = &imagesCollocatedWithBundle
		return i.DeepCopy(), nil
	}

	return newStuff, nil
}
func (i ImageRefs) LocalizeToBundle(imagesLockLocationConfig ImagesLockLocationConfig, relativeToRepo string) (ImageRefs, error) {
	locationsFound := true
	locationsConfig, err := imagesLockLocationConfig.Fetch()
	if err != nil {
		if _, ok := err.(*LocationsNotFound); !ok {
			return ImageRefs{}, err
		}

		locationsFound = false
	}

	imageRefs := ImageRefs{}

	for _, imgRef := range i.refs {
		imgRef := imgRef.DeepCopy()
		if locationsFound {
			for _, imgLoc := range locationsConfig.Images {
				if imgLoc.Image == imgRef.Image {
					isBundle := imgLoc.IsBundle
					imgRef.IsBundle = &isBundle
				}
			}
		}
		imageInBundleRepo := i.imageRelativeToBundle(imgRef.Image, relativeToRepo)
		imgRef.AddLocation(imageInBundleRepo)

		imageRefs.AddImagesRef(imgRef)
	}

	return imageRefs, nil
}
func (i ImageRefs) imageRelativeToBundle(img, relativeToRepo string) string {
	imgParts := strings.Split(img, "@")
	if len(imgParts) != 2 {
		panic(fmt.Sprintf("Internal inconsistency: The provided image URL '%s' does not contain a digest", img))
	}
	return relativeToRepo + "@" + imgParts[1]
}
func (i ImageRefs) imageRefsFound(imgRetriever ctlimg.ImagesMetadata) (ImageRefs, error) {
	if i.imagesCollocatedWithBundle != nil {
		return i.DeepCopy(), nil
	}

	imageRefs := ImageRefs{}
	imagesCollocatedWithBundle := true
	for _, imgRef := range i.refs {
		imgRef := imgRef.DeepCopy()
		if imgRef.IsBundle == nil {
			foundImg, err := imgRetriever.FirstImageExists(imgRef.Locations())
			if err != nil {
				return ImageRefs{}, err
			}

			// If cannot find the image in the bundle repo, will not localize any image
			// We assume that the bundle was not copied to the bundle location,
			// so there we cannot localize any image
			if foundImg != imgRef.PrimaryLocation() {
				imagesCollocatedWithBundle = false
				break
			}

			imgRef = imgRef.DiscardLocationsExcept(foundImg)
		}

		imageRefs.AddImagesRef(imgRef)
	}

	if !imagesCollocatedWithBundle {
		i.imagesCollocatedWithBundle = &imagesCollocatedWithBundle
		return i.DeepCopy(), nil
	}

	imageRefs.imagesCollocatedWithBundle = &imagesCollocatedWithBundle

	return imageRefs, nil
}
func (i ImageRefs) CollocatedWithBundle() bool {
	if i.imagesCollocatedWithBundle == nil {
		return false
	}
	return *i.imagesCollocatedWithBundle
}
func (i ImageRefs) ImagesLock() lockconfig.ImagesLock {
	imgLock := lockconfig.NewEmptyImagesLock()
	for _, ref := range i.refs {
		imgLock.AddImageRef(ref.ImageRef)
	}
	return imgLock
}
