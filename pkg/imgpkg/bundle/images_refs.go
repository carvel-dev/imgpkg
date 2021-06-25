// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 github.com/cppforlife/go-cli-ui/ui.UI

type ImageRef struct {
	lockconfig.ImageRef
	IsBundle *bool
}

func NewImageRef(imgRef lockconfig.ImageRef, isBundle bool) ImageRef {
	return ImageRef{ImageRef: imgRef, IsBundle: &isBundle}
}

func (i ImageRef) DeepCopy() ImageRef {
	return ImageRef{
		ImageRef: i.ImageRef.DeepCopy(),
		IsBundle: i.IsBundle,
	}
}
func (i *ImageRef) updateRepo(relativeToRepo string) *ImageRef {
	i.Image = i.replaceImageRepo(relativeToRepo)
	return i
}
func (i *ImageRef) replaceImageRepo(relativeToRepo string) string {
	imgParts := strings.Split(i.Image, "@")
	if len(imgParts) != 2 {
		panic(fmt.Sprintf("Internal inconsistency: The provided image URL '%s' does not contain a digest", i.Image))
	}
	updatedRepo := relativeToRepo + "@" + imgParts[1]
	return updatedRepo
}

type ImageRefs struct {
	refs                 []ImageRef
	imageLocationsConfig *ImageLocationsConfig
	originalImagesLock   lockconfig.ImagesLock

	*sync.Mutex
}

func NewEmptyImageRefs() ImageRefs {
	return ImageRefs{
		Mutex: &sync.Mutex{},
	}
}

func NewImageRefs(imagesLock lockconfig.ImagesLock, imagesLockLocationConfig ImagesLockLocationsConfig) (ImageRefs, error) {
	imageRefs := ImageRefs{
		Mutex:              &sync.Mutex{},
		originalImagesLock: imagesLock,
	}
	for _, lockImgRef := range imagesLock.Images {
		imageRefs.AddImagesRef(ImageRef{
			ImageRef: lockImgRef,
			IsBundle: nil,
		})
	}

	err := imageRefs.syncImageRefsWithLocationConfig(imagesLockLocationConfig)
	if err != nil {
		return ImageRefs{}, err
	}
	return imageRefs, nil
}

func (i *ImageRefs) LocalizeToBundle(relativeToRepo string) {
	for _, imgRef := range i.refs {
		imgRef.AddLocation(imgRef.replaceImageRepo(relativeToRepo))
		i.AddImagesRef(imgRef)
	}
}

func (i *ImageRefs) UpdateRelativeToRepo(imgRetriever ctlimg.ImagesMetadata, relativeToRepo string) (bool, error) {
	if i.imageLocationsConfig != nil {
		i.LocalizeToBundle(relativeToRepo)
		return true, nil
	}

	for _, ref := range i.refs {
		image, err := name.ParseReference(ref.updateRepo(relativeToRepo).Image)
		if err != nil {
			return false, err
		}
		_, err = imgRetriever.Digest(image)
		if err != nil {
			if terr, ok := err.(*transport.Error); ok {
				if _, ok := imageNotFoundStatusCode[terr.StatusCode]; ok {
					return false, nil
				}
			}
			return false, err
		}
	}

	i.LocalizeToBundle(relativeToRepo)

	return true, nil
}
func (i *ImageRefs) AddImagesRef(refs ...ImageRef) {
	i.Lock()
	defer i.Unlock()

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
func (i *ImageRefs) MarkAsBundle(image string, isBundle bool) {
	for j, ref := range i.refs {
		if ref.Image == image {
			i.refs[j].IsBundle = &isBundle
		}
	}
}
func (i ImageRefs) ImagesLock() lockconfig.ImagesLock {
	imgLock := lockconfig.NewEmptyImagesLock()
	for _, originalImg := range i.originalImagesLock.Images {
		ref, found := i.Find(originalImg.Image)
		if !found {
			panic(fmt.Errorf("Internal inconsistency: '%s' could not be found", originalImg.Image))
		}

		imgLock.Images = append(imgLock.Images, lockconfig.ImageRef{
			Image:       ref.PrimaryLocation(),
			Annotations: originalImg.Annotations,
		})
	}

	return imgLock
}
func (i ImageRefs) ImageRefs() []ImageRef {
	return i.refs
}
func (i *ImageRefs) syncImageRefsWithLocationConfig(imagesLockLocationConfig ImagesLockLocationsConfig) error {
	locationsConfig, err := imagesLockLocationConfig.Config()
	if _, ok := err.(*LocationsNotFound); ok {
		return nil
	}

	if err != nil {
		return err
	}

	for _, imgLoc := range locationsConfig.Images {
		i.MarkAsBundle(imgLoc.Image, imgLoc.IsBundle)
	}
	i.imageLocationsConfig = &locationsConfig

	return nil
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesLockLocationConfig
type ImagesLockLocationsConfig interface {
	Config() (ImageLocationsConfig, error)
}
