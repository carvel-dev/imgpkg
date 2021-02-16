// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	regname "github.com/google/go-containerregistry/pkg/name"
)

// ResolvedImage respresents an image that will be resolved into url+digest
type ResolvedImage struct {
	url            string
	imagesMetadata ImagesMetadata
}

type ResolvedImageSourceURL struct {
	Type string // always set to 'resolved'
	URL  string
	Tag  string
}

func NewResolvedImage(url string, imagesMetadata ImagesMetadata) ResolvedImage {
	return ResolvedImage{url, imagesMetadata}
}

func (i ResolvedImage) URL() (string, error) {
	tag, err := regname.NewTag(i.url, regname.WeakValidation)
	if err != nil {
		return "", err
	}

	hash, err := i.imagesMetadata.Digest(tag)
	if err != nil {
		return "", err
	}

	digest, err := regname.NewDigest(tag.Repository.String() + "@" + hash.String())
	if err != nil {
		return "", err
	}

	return digest.Name(), nil
}
