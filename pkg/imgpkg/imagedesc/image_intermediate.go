// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// ImageIndexIntermediate struct for ImageIndex to help in convertion to ImagIndexWithRef struct for oci image workflow..
type ImageIndexIntermediate struct {
	Index regv1.ImageIndex
	ref   string
	tag   string
}

// ImageIntermediate struct for Image to help in convertion to ImageWithRef struct for oci image workflow.
type ImageIntermediate struct {
	Image regv1.Image
	ref   string
	tag   string
}

// Ref returns the reference in the imageintermediate struct.
func (mi ImageIntermediate) Ref() string {
	return mi.ref
}

// SetRef sets the reference in the imageintermediate struct.
func (mi *ImageIntermediate) SetRef(ref string) {
	mi.ref = ref
}

// Tag returns the tag value set in the imageintermediate struct.
func (mi ImageIntermediate) Tag() string {
	return mi.tag
}

// SetTag sets the tag value in the imageintermediate struct.
func (mi *ImageIntermediate) SetTag(tag string) {
	mi.tag = tag
}

// Layers returns the ordered collection of filesystem layers that comprise this image.
func (mi ImageIntermediate) Layers() ([]regv1.Layer, error) {
	return mi.Image.Layers()
}
// MediaType of the image's manifest.
func (mi ImageIntermediate) MediaType() (types.MediaType, error) {
	return mi.Image.MediaType()
}

// Size returns the size of the image.
func (mi ImageIntermediate) Size() (int64, error) {
	return mi.Image.Size()
}

// ConfigName returns the name of the image's configuration.
func (mi ImageIntermediate) ConfigName() (regv1.Hash, error) {
	return mi.Image.ConfigName()
}

// ConfigFile returns the image's config file.
func (mi ImageIntermediate) ConfigFile() (*regv1.ConfigFile, error) {
	return mi.Image.ConfigFile()
}

// RawConfigFile returns the serialized bytes of ConfigFile().
func (mi ImageIntermediate) RawConfigFile() ([]byte, error) {
	return mi.Image.RawConfigFile()
}

// Digest returns the sha256 of this image's manifest.
func (mi ImageIntermediate) Digest() (regv1.Hash, error) {
	return mi.Image.Digest()
}

// Manifest returns this image's Manifest object.
func (mi ImageIntermediate) Manifest() (*regv1.Manifest, error) {
	return mi.Image.Manifest()
}

// RawManifest returns the serialized bytes of Manifest()
func (mi ImageIntermediate) RawManifest() ([]byte, error) {
	return mi.Image.RawManifest()
}

// LayerByDigest returns a Layer for interacting with a particular layer of the image, looking it up by "digest" (the compressed hash).
func (mi ImageIntermediate) LayerByDigest(h regv1.Hash) (regv1.Layer, error) {
	return mi.Image.LayerByDigest(h)
}

// LayerByDiffID is an analog to LayerByDigest, looking up by "diff id" (the uncompressed hash).
func (mi ImageIntermediate) LayerByDiffID(h regv1.Hash) (regv1.Layer, error) {
	return mi.Image.LayerByDiffID(h)
}

// Ref returns the reference in the imageindexintermediate struct.
func (mi ImageIndexIntermediate) Ref() string {
	return mi.ref
}

// SetRef sets the reference in the imageindexintermediate struct.
func (mi *ImageIndexIntermediate) SetRef(ref string) {
	mi.ref = ref
}

// Tag returns the tag value set in the imageindexintermediate struct.
func (mi ImageIndexIntermediate) Tag() string {
	return mi.tag
}

// SetTag sets the tag value in the imageindexintermediate struct.
func (mi *ImageIndexIntermediate) SetTag(tag string) {
	mi.tag = tag
}

// MediaType of the imageindex's manifest.
func (mi ImageIndexIntermediate) MediaType() (types.MediaType, error) {
	return mi.Index.MediaType()
}

// Digest returns the sha256 of this imageindex's manifest.
func (mi ImageIndexIntermediate) Digest() (regv1.Hash, error) {
	return mi.Index.Digest()
}

// Size returns the size of the imageindex.
func (mi ImageIndexIntermediate) Size() (int64, error) {
	return mi.Index.Size()
}

// IndexManifest returns this image index's manifest object.
func (mi ImageIndexIntermediate) IndexManifest() (*regv1.IndexManifest, error) {
	return mi.Index.IndexManifest()
}

// RawManifest returns the serialized bytes of IndexManifest().
func (mi ImageIndexIntermediate) RawManifest() ([]byte, error) {
	return mi.Index.RawManifest()
}

// Image returns a v1.Image that this ImageIndex references.
func (mi ImageIndexIntermediate) Image(h regv1.Hash) (regv1.Image, error) {
	return mi.Index.Image(h)
}

// ImageIndex returns a v1.ImageIndex that this ImageIndex references.
func (mi ImageIndexIntermediate) ImageIndex(h regv1.Hash) (regv1.ImageIndex, error) {
	return mi.Index.ImageIndex(h)
}

var _ regv1.ImageIndex = ImageIndexIntermediate{}
var _ regv1.Image = ImageIntermediate{}
