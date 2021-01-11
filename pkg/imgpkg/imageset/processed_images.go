// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imageset

import (
	"fmt"
	"sort"
	"sync"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
)

type ProcessedImage struct {
	UnprocessedImageRef
	DigestRef string

	Image      regv1.Image
	ImageIndex regv1.ImageIndex
}

type ProcessedImages struct {
	imgs     map[UnprocessedImageRef]ProcessedImage
	imgsLock sync.Mutex
}

func (i ProcessedImage) Validate() {
	_, err := regname.NewDigest(i.DigestRef)
	if err != nil {
		panic(fmt.Sprintf("Digest need to be provided: %s", err))
	}

	if i.Image == nil && i.ImageIndex == nil {
		panic("Either Image or ImageIndex must be provided")
	}

	i.UnprocessedImageRef.Validate()
}

func NewProcessedImages() *ProcessedImages {
	return &ProcessedImages{imgs: map[UnprocessedImageRef]ProcessedImage{}}
}

func (i *ProcessedImages) Add(img ProcessedImage) {
	i.imgsLock.Lock()
	defer i.imgsLock.Unlock()

	img.Validate()
	i.imgs[img.UnprocessedImageRef] = img
}

func (i *ProcessedImages) FindByURL(unprocessedImageURL UnprocessedImageRef) (ProcessedImage, bool) {
	i.imgsLock.Lock()
	defer i.imgsLock.Unlock()

	img, found := i.imgs[unprocessedImageURL]
	return img, found
}

func (i *ProcessedImages) All() []ProcessedImage {
	i.imgsLock.Lock()
	defer i.imgsLock.Unlock()

	var result []ProcessedImage
	for _, img := range i.imgs {
		result = append(result, img)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UnprocessedImageRef.DigestRef < result[j].UnprocessedImageRef.DigestRef
	})
	return result
}
