// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"strings"

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

// BuildLegibleUploadTagRef Builds a legible tag from original image reference
func BuildLegibleUploadTagRef(item WithDigest, itemRef string, importRepo regname.Repository) (regname.Tag, error) {
	digest, err := item.Digest()
	if err != nil {
		return regname.Tag{}, err
	}
	itemDigest, err := regname.NewDigest(itemRef)
	if err != nil {
		return regname.Tag{}, err
	}

	origRepoPath := itemDigest.Context().RepositoryStr()
	tagStartIdx := len(origRepoPath) - 49
	if tagStartIdx < 0 {
		tagStartIdx = 0
	}

	dashedRepo := fmt.Sprintf("%s-%s-%s.imgpkg", origRepoPath[tagStartIdx:], digest.Algorithm, digest.Hex)
	tag := strings.ReplaceAll(dashedRepo, "/", "-")
	uploadTagRef, err := regname.NewTag(fmt.Sprintf("%s:%s", importRepo.Name(), tag))

	return uploadTagRef, err
}
