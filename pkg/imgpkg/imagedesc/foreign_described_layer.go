// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	"fmt"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageutils/gzip"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageutils/verify"
	"io"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type ForeignDescribedLayer struct {
	desc     ImageLayerDescriptor
	contents LayerContents
}

var _ regv1.Layer = ForeignDescribedLayer{}

func NewForeignDescribedLayer(desc ImageLayerDescriptor, contents LayerContents) ForeignDescribedLayer {
	return ForeignDescribedLayer{desc, contents}
}

func (l ForeignDescribedLayer) Digest() (regv1.Hash, error) { return regv1.NewHash(l.desc.Digest) }
func (l ForeignDescribedLayer) DiffID() (regv1.Hash, error) { return regv1.NewHash(l.desc.DiffID) }

func (l ForeignDescribedLayer) Compressed() (io.ReadCloser, error) {
	return l.contents.Open()
}

func (l ForeignDescribedLayer) Uncompressed() (io.ReadCloser, error) {
	rc, err := l.contents.Open()
	if err != nil {
		return nil, err
	}

	h, err := l.Digest()
	if err != nil {
		return nil, fmt.Errorf("Computing digest: %v", err)
	}

	rc, err = verify.ReadCloser(rc, h)
	if err != nil {
		return nil, fmt.Errorf("Creating verified reader: %v", err)
	}

	return gzip.ReadCloser(rc), nil
}

func (l ForeignDescribedLayer) Size() (int64, error) { return l.desc.Size, nil }

func (l ForeignDescribedLayer) MediaType() (types.MediaType, error) {
	return types.MediaType(l.desc.MediaType), nil
}
