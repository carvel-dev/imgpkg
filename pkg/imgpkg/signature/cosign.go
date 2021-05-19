// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package signature

import (
	"fmt"
	"net/http"

	regname "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/registry"
	"github.com/k14s/imgpkg/pkg/imgpkg/signature/cosign"
)

func NewCosign(reg registry.Registry) *Cosign {
	return &Cosign{registry: reg}
}

type Cosign struct {
	registry registry.Registry
}

func (c Cosign) Signature(imageRef regname.Digest) (imageset.UnprocessedImageRef, error) {
	sigTagRef, err := c.signatureTag(imageRef)
	if err != nil {
		return imageset.UnprocessedImageRef{}, err
	}

	sigDigest, err := c.registry.Digest(sigTagRef)
	if err != nil {
		if transportErr, ok := err.(*transport.Error); ok {
			if transportErr.StatusCode == http.StatusNotFound {
				return imageset.UnprocessedImageRef{}, NotFound{}
			}
		}
		return imageset.UnprocessedImageRef{}, err
	}

	sigDigestRef := imageRef.Digest(sigDigest.String())
	return imageset.UnprocessedImageRef{DigestRef: sigDigestRef.Name(), Tag: sigTagRef.TagStr()}, nil
}

func (c Cosign) signatureTag(reference regname.Digest) (regname.Tag, error) {
	digest, err := v1.NewHash(reference.DigestStr())
	if err != nil {
		return regname.Tag{}, fmt.Errorf("Converting to hash: %s", err)
	}
	return regname.NewTag(reference.Repository.Name() + ":" + cosign.Munge(v1.Descriptor{Digest: digest}))
}
