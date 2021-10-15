// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package signature

import (
	"fmt"
	"net/http"

	regname "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/signature/cosign"
)

type Cosign struct {
	registry registry.Registry
}

func NewCosign(reg registry.Registry) *Cosign {
	return &Cosign{registry: reg}
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
				return imageset.UnprocessedImageRef{}, NotFoundErr{}
			}
		}
		return imageset.UnprocessedImageRef{}, err
	}

	return imageset.UnprocessedImageRef{
		DigestRef: imageRef.Digest(sigDigest.String()).Name(),
		Tag:       sigTagRef.TagStr(),
	}, nil
}

func (c Cosign) signatureTag(reference regname.Digest) (regname.Tag, error) {
	digest, err := v1.NewHash(reference.DigestStr())
	if err != nil {
		return regname.Tag{}, fmt.Errorf("Converting to hash: %s", err)
	}
	return regname.NewTag(reference.Repository.Name() + ":" + cosign.Munge(v1.Descriptor{Digest: digest}))
}
