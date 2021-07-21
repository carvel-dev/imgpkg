// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package plainimage

import (
	"fmt"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
)

type ImagesDescriptor interface {
	Get(regname.Reference) (*regremote.Descriptor, error)
	Head(regname.Reference) (*regv1.Descriptor, error)
}

type PlainImage struct {
	ref      string
	registry ImagesDescriptor

	parsedRef    regname.Reference
	parsedDigest string

	fetchedImage regv1.Image
}

func MustNewPlainImage(ref string, imagesDescriptor ImagesDescriptor) (*PlainImage, error) {
	err := assertPlainImageRefIsAnImage(ref, imagesDescriptor)
	if err != nil {
		return nil, err
	}

	return NewPlainImage(ref, imagesDescriptor), nil
}

func NewPlainImage(ref string, imagesDescriptor ImagesDescriptor) *PlainImage {
	return &PlainImage{ref: ref, registry: imagesDescriptor}
}

func assertPlainImageRefIsAnImage(ref string, imagesDescriptor ImagesDescriptor) error {
	plainImageRef, err := regname.ParseReference(ref)
	if err != nil {
		return err
	}

	imageDescriptor, err := imagesDescriptor.Head(plainImageRef)
	if err != nil {
		return err
	}

	if !imageDescriptor.MediaType.IsImage() {
		return fmt.Errorf("Expected an Image but got: %s", string(imageDescriptor.MediaType))
	}

	return nil
}

func NewFetchedPlainImageWithTag(digestRef string, tag string, fetchedImage regv1.Image) *PlainImage {
	parsedDigestRef, err := regname.NewDigest(digestRef)
	if err != nil {
		panic(fmt.Sprintf("Expected valid Digest Ref: %s", err))
	}

	var parsedRef regname.Reference
	if tag == "" {
		parsedRef = parsedDigestRef
	} else {
		parsedRef, err = regname.NewTag(parsedDigestRef.Context().Name() + ":" + tag)
		if err != nil {
			panic(fmt.Sprintf("Expected valid Tag Ref: %s", err))
		}
	}

	return &PlainImage{
		parsedRef:    parsedRef,
		parsedDigest: parsedDigestRef.DigestStr(),
		fetchedImage: fetchedImage,
	}
}

func (i *PlainImage) Repo() string {
	if i.parsedRef == nil {
		panic("Unexpected usage of Repo(); call Fetch before")
	}
	return i.parsedRef.Context().Name()
}

func (i *PlainImage) DigestRef() string {
	if i.parsedRef == nil {
		panic("Unexpected usage of DigestRef(); call Fetch before")
	}
	if len(i.parsedDigest) == 0 {
		panic("Unexpected usage of DigestRef(); call Fetch before")
	}
	return i.parsedRef.Context().Name() + "@" + i.parsedDigest
}

func (i *PlainImage) Tag() string {
	if i.parsedRef == nil {
		panic("Unexpected usage of Tag(); call Fetch before")
	}
	if tagRef, ok := i.parsedRef.(regname.Tag); ok {
		return tagRef.TagStr()
	}
	return "" // was a digest ref, so no tag
}

func (i *PlainImage) Fetch() (regv1.Image, error) {
	var err error
	if i.fetchedImage != nil {
		return i.fetchedImage, nil
	}

	i.parsedRef, err = regname.ParseReference(i.ref, regname.WeakValidation)
	if err != nil {
		return nil, err
	}

	imgDescriptor, err := i.registry.Get(i.parsedRef)
	if err != nil {
		return nil, fmt.Errorf("Fetching image: %s", err)
	}

	if !imgDescriptor.MediaType.IsImage() {
		return nil, notAnImageError{}
	}

	i.fetchedImage, err = imgDescriptor.Image()
	if err != nil {
		return nil, fmt.Errorf("Fetching image: %s", err)
	}

	digest, err := i.fetchedImage.Digest()
	if err != nil {
		return nil, fmt.Errorf("Getting image digest: %s", err)
	}

	i.parsedDigest = digest.String()

	return i.fetchedImage, nil
}

func (i *PlainImage) Pull(outputPath string, ui ui.UI) error {
	img, err := i.Fetch()
	if err != nil {
		return err
	}

	if img == nil {
		panic("Not supported Pull on pre fetched PlainImage")
	}

	ui.BeginLinef("Pulling image '%s'\n", i.DigestRef())

	err = ctlimg.NewDirImage(outputPath, img, ui).AsDirectory()
	if err != nil {
		return fmt.Errorf("Extracting image into directory: %s", err)
	}

	return nil
}

func IsNotAnImageError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(notAnImageError)
	return ok
}

type notAnImageError struct {
}

func (n notAnImageError) Error() string {
	return "Not an image"
}
