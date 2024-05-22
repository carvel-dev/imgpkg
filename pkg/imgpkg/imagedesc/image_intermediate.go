// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// intermediate struct for Image and ImageIndex to convert to imagwithref type for oci image
type ImageIndexIntermediate struct {
	Index regv1.ImageIndex
	ref   string
	tag   string
}

type ImageIntermediate struct {
	Image regv1.Image
	ref   string
	tag   string
}

func (mi ImageIntermediate) Ref() string {
	return mi.ref
}

func (mi *ImageIntermediate) SetRef(ref string) {
	mi.ref = ref
}

func (mi ImageIntermediate) Tag() string {
	return mi.tag
}

func (mi *ImageIntermediate) SetTag(tag string) {
	mi.tag = tag
}

func (mi ImageIntermediate) Layers() ([]regv1.Layer, error) {
	return mi.Image.Layers()
}

func (mi ImageIntermediate) MediaType() (types.MediaType, error) {
	return mi.Image.MediaType()
}

func (mi ImageIntermediate) Size() (int64, error) {
	return mi.Image.Size()
}

func (mi ImageIntermediate) ConfigName() (regv1.Hash, error) {
	return mi.Image.ConfigName()
}

func (mi ImageIntermediate) ConfigFile() (*regv1.ConfigFile, error) {
	return mi.Image.ConfigFile()
}

func (mi ImageIntermediate) RawConfigFile() ([]byte, error) {
	return mi.Image.RawConfigFile()
}

func (mi ImageIntermediate) Digest() (regv1.Hash, error) {
	return mi.Image.Digest()
}

func (mi ImageIntermediate) Manifest() (*regv1.Manifest, error) {
	return mi.Image.Manifest()
}

func (mi ImageIntermediate) RawManifest() ([]byte, error) {
	return mi.Image.RawManifest()
}

func (mi ImageIntermediate) LayerByDigest(h regv1.Hash) (regv1.Layer, error) {
	return mi.Image.LayerByDigest(h)
}

func (mi ImageIntermediate) LayerByDiffID(h regv1.Hash) (regv1.Layer, error) {
	return mi.Image.LayerByDiffID(h)
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
var _ regv1.Image = ImageIntermediate{}
