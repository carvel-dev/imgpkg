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
func BuildLegibleUploadTagRef(itemRef string, importRepo regname.Repository) (regname.Tag, error) {
	// ex.:
	// 1. registry.f.q.d.n/myorg/my-img@sha256:012345679e2a06317dc02abcde123759eb9e6105a0731224a0c63898abcde123
	// 2. registry.f.q.d.n/myorg/my-img:some-tag@sha256:012345679e2a06317dc02abcde123759eb9e6105a0731224a0c63898abcde123
	origRepoAndTag := strings.Split(itemRef, "@")
	origRepoPath := strings.Split(origRepoAndTag[0], "/")
	origImgOrg := origRepoPath[len(origRepoPath)-2]
	origImgName := origRepoPath[len(origRepoPath)-1]
	tag := origImgOrg + "-" + origImgName + "-" +
		// 01234567
		strings.Split(origRepoAndTag[1], ":")[1][:8]
	tag = strings.Replace(tag, ":", "-", 1)
	uploadTagRef, err := regname.NewTag(fmt.Sprintf("%s:%s", importRepo.Name(), tag))
	return uploadTagRef, err
}

