// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"fmt"

	regname "github.com/google/go-containerregistry/pkg/name"
)

// ResolvedImage respresents an image that will be resolved into url+digest
type ResolvedImage struct {
	url      string
	registry Registry
}

type ResolvedImageSourceURL struct {
	Type string // always set to 'resolved'
	URL  string
	Tag  string
}

func NewResolvedImage(url string, registry Registry) ResolvedImage {
	return ResolvedImage{url, registry}
}

func (i ResolvedImage) URL() (string, error) {
	tag, err := regname.NewTag(i.url, regname.WeakValidation)
	if err != nil {
		return "", err
	}

	imgDescriptor, err := i.registry.Generic(tag)
	if err != nil {
		return "", err
	}

	// Resolve image second time because some older registry can
	// return "random" digests that change for every request.
	// See https://github.com/k14s/imgpkg/issues/21 for details.
	imgDescriptor2, err := i.registry.Generic(tag)
	if err != nil {
		return "", err
	}

	if imgDescriptor.Digest.String() != imgDescriptor2.Digest.String() {
		return "", fmt.Errorf("Expected digest resolution to be consistent over two separate requests")
	}

	digestURL, err := regname.NewDigest(tag.Repository.String() + "@" + imgDescriptor.Digest.String())
	if err != nil {
		return "", err
	}

	return digestURL.Name(), nil
}
