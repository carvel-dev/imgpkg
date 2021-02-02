// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"fmt"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"strings"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regtran "github.com/google/go-containerregistry/pkg/v1/remote/transport"
	regtypes "github.com/google/go-containerregistry/pkg/v1/types"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesMetadata
type ImagesMetadata interface {
	Generic(regname.Reference) (regv1.Descriptor, error)
	Index(regname.Reference) (regv1.ImageIndex, error)
	Image(regname.Reference) (regv1.Image, error)
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesReaderWriter
type ImagesReaderWriter interface {
	ImagesMetadata
	WriteImage(regname.Reference, regv1.Image) error
	WriteIndex(regname.Reference, regv1.ImageIndex) error
	WriteTag(regname.Tag, regremote.Taggable) error
}

type Images struct {
	ref      regname.Reference
	metadata ImagesMetadata
}

func NewImages(ref regname.Reference, metadata ImagesMetadata) Images {
	return Images{ref: ref, metadata: errImagesMetadata{metadata}}
}

func (tds Images) Images() ([]regv1.Image, error) {
	desc, err := tds.metadata.Generic(tds.ref)
	if err != nil {
		return nil, err
	}

	var result []regv1.Image

	if tds.isImageIndex(desc) {
		imgs, err := tds.buildImageIndex(tds.ref)
		if err != nil {
			return nil, err
		}
		result = append(result, imgs...)
	} else {
		img, err := tds.buildImage(tds.ref)
		if err != nil {
			return nil, err
		}
		result = append(result, img)
	}

	return result, nil
}

func (tds Images) buildImageIndex(ref regname.Reference) ([]regv1.Image, error) {
	imgIndex, err := tds.metadata.Index(ref)
	if err != nil {
		return nil, err
	}

	imgIndexManifest, err := imgIndex.IndexManifest()
	if err != nil {
		return nil, err
	}

	var result []regv1.Image

	for _, manDesc := range imgIndexManifest.Manifests {
		if tds.isImageIndex(manDesc) {
			imgs, err := tds.buildImageIndex(tds.buildRef(ref, manDesc.Digest.String()))
			if err != nil {
				return nil, err
			}
			result = append(result, imgs...)
		} else {
			img, err := tds.buildImage(tds.buildRef(ref, manDesc.Digest.String()))
			if err != nil {
				return nil, err
			}
			result = append(result, img)
		}
	}

	return result, nil
}

func (tds Images) buildImage(ref regname.Reference) (regv1.Image, error) {
	return tds.metadata.Image(ref)
}

func (Images) isImageIndex(desc regv1.Descriptor) bool {
	switch desc.MediaType {
	case regtypes.OCIImageIndex, regtypes.DockerManifestList:
		return true
	}
	return false
}

func (tds Images) buildRef(otherRef regname.Reference, digest string) regname.Reference {
	newRef, err := regname.NewDigest(fmt.Sprintf("%s@%s", otherRef.Context().Name(), digest))
	if err != nil {
		panic(fmt.Sprintf("Building new ref"))
	}
	return newRef
}

type errImagesMetadata struct {
	delegate ImagesMetadata
}

func (m errImagesMetadata) Generic(ref regname.Reference) (regv1.Descriptor, error) {
	desc, err := m.delegate.Generic(ref)
	return desc, m.betterErr(ref, err)
}

func (m errImagesMetadata) Index(ref regname.Reference) (regv1.ImageIndex, error) {
	idx, err := m.delegate.Index(ref)
	return idx, m.betterErr(ref, err)
}

func (m errImagesMetadata) Image(ref regname.Reference) (regv1.Image, error) {
	img, err := m.delegate.Image(ref)
	return img, m.betterErr(ref, err)
}

func (m errImagesMetadata) betterErr(ref regname.Reference, err error) error {
	if err != nil {
		if strings.Contains(err.Error(), string(regtran.ManifestUnknownErrorCode)) {
			err = fmt.Errorf("Encountered an error most likely because this image is in Docker Registry v1 format; only v2 or OCI image format is supported (underlying error: %s)", err)
		}
		err = fmt.Errorf("Working with %s: %s", ref.Name(), err)
	}
	return err
}
