// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package artifacts

import (
	"net/http"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/artifacts/cosign"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset"
)

// DigestReader Interface that knows how to read a Digest from a registry
type DigestReader interface {
	Digest(reference regname.Reference) (regv1.Hash, error)
}

// Cosign Signature retriever
type Cosign struct {
	registry DigestReader
}

// NewCosign constructor for Signature retriever
func NewCosign(reg DigestReader) *Cosign {
	return &Cosign{registry: reg}
}

// Signature retrieves the Image information that contains the signature for the provided Image
func (c Cosign) Signature(imageRef regname.Digest) (imageset.UnprocessedImageRef, error) {
	sigTagRef, err := cosign.SignatureTag(imageRef)
	if err != nil {
		return imageset.UnprocessedImageRef{}, err
	}

	return c.findArtifact(imageRef, err, sigTagRef)
}

// SBOM retrieves the Image information that contains the signature for the provided Image
func (c Cosign) SBOM(imageRef regname.Digest) (imageset.UnprocessedImageRef, error) {
	sigTagRef, err := cosign.SBOMTag(imageRef)
	if err != nil {
		return imageset.UnprocessedImageRef{}, err
	}

	return c.findArtifact(imageRef, err, sigTagRef)
}

// Attestation retrieves the Image information that contains the signature for the provided Image
func (c Cosign) Attestation(imageRef regname.Digest) (imageset.UnprocessedImageRef, error) {
	sigTagRef, err := cosign.AttestationTag(imageRef)
	if err != nil {
		return imageset.UnprocessedImageRef{}, err
	}

	return c.findArtifact(imageRef, err, sigTagRef)
}

func (c Cosign) findArtifact(imageRef regname.Digest, err error, sigTagRef regname.Tag) (imageset.UnprocessedImageRef, error) {
	sigDigest, err := c.registry.Digest(sigTagRef)
	if err != nil {
		if transportErr, ok := err.(*transport.Error); ok {
			if transportErr.StatusCode == http.StatusNotFound {
				return imageset.UnprocessedImageRef{}, NotFoundErr{}
			}
			if transportErr.StatusCode == http.StatusForbidden {
				return imageset.UnprocessedImageRef{}, AccessDeniedErr{imageRef: sigTagRef.Identifier()}
			}
		}
		return imageset.UnprocessedImageRef{}, err
	}

	return imageset.UnprocessedImageRef{
		DigestRef: imageRef.Digest(sigDigest.String()).Name(),
		Tag:       sigTagRef.TagStr(),
	}, nil
}
