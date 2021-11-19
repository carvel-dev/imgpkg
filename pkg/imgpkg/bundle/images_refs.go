// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
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

type ImageRefs struct {
	refs                 []ImageRef
	imageLocationsConfig *ImageLocationsConfig
	originalImagesLock   *lockconfig.ImagesLock

	refsLock *sync.Mutex
}

func NewImageRefs() ImageRefs {
	return ImageRefs{
		refsLock: &sync.Mutex{},
	}
}

// NewImageRefsFromImagesLock Create a new ImageRefs from the provided lockconfig.ImagesLock and ImageLocationsConfig
func NewImageRefsFromImagesLock(imagesLock lockconfig.ImagesLock, imageRefsLocationConfig ImageRefLocationsConfig) (ImageRefs, error) {
	imageRefs := ImageRefs{
		refsLock:           &sync.Mutex{},
		originalImagesLock: &imagesLock,
	}
	for _, lockImgRef := range imagesLock.Images {
		imageRefs.AddImagesRef(ImageRef{
			ImageRef: lockImgRef,
			IsBundle: nil,
		})
	}

	err := imageRefs.syncImageRefsWithLocationConfig(imageRefsLocationConfig)
	if err != nil {
		return ImageRefs{}, err
	}
	return imageRefs, nil
}

func (i *ImageRefs) LocalizeToRepo(relativeToRepo string) {
	i.refsLock.Lock()
	defer i.refsLock.Unlock()

	for j, imgRef := range i.refs {
		i.refs[j].AddLocation(replaceImageRepo(imgRef.Image, relativeToRepo))
	}
}

func (i *ImageRefs) UpdateRelativeToRepo(imgRetriever ImagesMetadata, relativeToRepo string) (bool, error) {
	if i.imageLocationsConfig != nil {
		i.LocalizeToRepo(relativeToRepo)
		return true, nil
	}

	for _, ref := range i.refs {
		image, err := name.ParseReference(replaceImageRepo(ref.Image, relativeToRepo))
		if err != nil {
			return false, err
		}
		_, err = imgRetriever.Digest(image)
		if err != nil {
			if terr, ok := err.(*transport.Error); ok {
				if i.imageIsNotFound(terr) {
					return false, nil
				}
			}
			return false, err
		}
	}

	i.LocalizeToRepo(relativeToRepo)

	return true, nil
}

func (i *ImageRefs) AddImagesRef(refs ...ImageRef) {
	i.refsLock.Lock()
	defer i.refsLock.Unlock()

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
			i.refs = append(i.refs, ref.DeepCopy())
		}
	}
}

func (i *ImageRefs) Find(ref string) (ImageRef, bool) {
	i.refsLock.Lock()
	defer i.refsLock.Unlock()

	for _, imageRef := range i.refs {
		if imageRef.Image == ref {
			return imageRef.DeepCopy(), true
		}
	}

	return ImageRef{}, false
}

func (i *ImageRefs) MarkAsBundle(image string, isBundle bool) {
	i.refsLock.Lock()
	defer i.refsLock.Unlock()

	for j, ref := range i.refs {
		if ref.Image == image {
			i.refs[j].IsBundle = &isBundle
		}
	}
}

func (i ImageRefs) ImagesLock() lockconfig.ImagesLock {
	if i.originalImagesLock == nil {
		panic("Internal inconsistency: ImagesLock was not provided")
	}

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
	var refsCopy []ImageRef
	for _, ref := range i.refs {
		refsCopy = append(refsCopy, ref.DeepCopy())
	}
	return refsCopy
}

func (i *ImageRefs) imageIsNotFound(terr *transport.Error) bool {
	_, ok := imageNotFoundStatusCode[terr.StatusCode]
	return ok
}

func (i *ImageRefs) syncImageRefsWithLocationConfig(imagesLockLocationConfig ImageRefLocationsConfig) error {
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

func replaceImageRepo(imageName string, relativeToRepo string) string {
	imgParts := strings.Split(imageName, "@")
	if len(imgParts) != 2 {
		panic(fmt.Sprintf("Internal inconsistency: The provided image URL '%s' does not contain a digest", imageName))
	}
	updatedRepo := relativeToRepo + "@" + imgParts[1]
	return updatedRepo
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImageRefLocationsConfig
type ImageRefLocationsConfig interface {
	Config() (ImageLocationsConfig, error)
}
