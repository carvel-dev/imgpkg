// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
)

// WithDigest are items that Digest() can be called on
type WithDigest interface {
	Digest() (regv1.Hash, error)
}

// BuildDefaultUploadTagRef Builds a tag from the digest Algorithm and Digest
func BuildDefaultUploadTagRef(item WithDigest, importRepo regname.Repository) (regname.Tag, error) {
	digest, err := item.Digest()
	if err != nil {
		return regname.Tag{}, err
	}

	tag := fmt.Sprintf("%s-%s.imgpkg", digest.Algorithm, digest.Hex)
	uploadTagRef, err := regname.NewTag(fmt.Sprintf("%s:%s", importRepo.Name(), tag))
	if err != nil {
		return regname.Tag{}, fmt.Errorf("building default upload tag image ref: %s", err)
	}
	return uploadTagRef, nil
}
