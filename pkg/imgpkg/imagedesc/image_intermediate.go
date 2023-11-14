// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type ImageIndexIntermediate struct {
	Index regv1.ImageIndex
	ref   string
	tag   string
}

func (mi ImageIndexIntermediate) Ref() string {
	return mi.ref
}

func (mi *ImageIndexIntermediate) SetRef(ref string) {
	mi.ref = ref
}

func (mi ImageIndexIntermediate) Tag() string {
	return mi.tag
}

func (mi *ImageIndexIntermediate) SetTag(tag string) {
	mi.tag = tag
}

func (mi ImageIndexIntermediate) MediaType() (types.MediaType, error) {
	return mi.Index.MediaType()
}

func (mi ImageIndexIntermediate) Digest() (regv1.Hash, error) {
	return mi.Index.Digest()
}

func (mi ImageIndexIntermediate) Size() (int64, error) {
	return mi.Index.Size()
}

func (mi ImageIndexIntermediate) IndexManifest() (*regv1.IndexManifest, error) {
	return mi.Index.IndexManifest()
}

func (mi ImageIndexIntermediate) RawManifest() ([]byte, error) {
	return mi.Index.RawManifest()
}

func (mi ImageIndexIntermediate) Image(h regv1.Hash) (regv1.Image, error) {
	return mi.Index.Image(h)
}

func (mi ImageIndexIntermediate) ImageIndex(h regv1.Hash) (regv1.ImageIndex, error) {
	return mi.Index.ImageIndex(h)
}

var _ regv1.ImageIndex = ImageIndexIntermediate{}
