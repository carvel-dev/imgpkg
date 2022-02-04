// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package plainimage

import (
	"fmt"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	ctlimg "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/image"
)

type ImagesDescriptor interface {
	Get(regname.Reference) (*regremote.Descriptor, error)
}

type PlainImage struct {
	imagesDescriptor ImagesDescriptor

	unparsedRef  string
	parsedRef    regname.Reference
	parsedDigest string

	fetchedImage regv1.Image
}

func NewPlainImage(ref string, imgDescriptor ImagesDescriptor) *PlainImage {
	return &PlainImage{unparsedRef: ref, imagesDescriptor: imgDescriptor}
}

func NewFetchedPlainImageWithTag(digestRef string, tag string, fetchedImage regv1.Image) *PlainImage {
	if fetchedImage == nil {
		panic("Expected a pre-fetched image")
	}

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

// DigestRef Image full location including registry, repository and digest
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

	i.parsedRef, err = regname.ParseReference(i.unparsedRef, regname.WeakValidation)
	if err != nil {
		return nil, err
	}

	imgDescriptor, err := i.imagesDescriptor.Get(i.parsedRef)
	if err != nil {
		return nil, fmt.Errorf("Fetching image: %s", err)
	}

	if !imgDescriptor.MediaType.IsImage() {
		i.parsedDigest = imgDescriptor.Digest.String()
		return nil, notAnImageError{imgDescriptor.MediaType}
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

func (i *PlainImage) IsImage() (bool, error) {
	img, err := i.Fetch()
	if img == nil && err == nil {
		panic("Unreachable code")
	}

	if err != nil {
		if IsNotAnImageError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
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
	mediaType types.MediaType
}

func (n notAnImageError) Error() string {
	return fmt.Sprintf("Expected an Image but got: %s", n.mediaType)
}
