// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	"fmt"
	"io"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	regv1types "github.com/google/go-containerregistry/pkg/v1/types"
)

type RegistryRemoteImage interface {
	Get(reference regname.Reference) (*remote.Descriptor, error)
}

type ImageOrIndex struct {
	Image *ImageWithRef
	Index *ImageIndexWithRef
}

type ImageWithRef interface {
	regv1.Image
	Ref() string
	Tag() string
}

type ImageIndexWithRef interface {
	regv1.ImageIndex
	Ref() string
	Tag() string
}

type LayerProvider interface {
	FindLayer(ImageLayerDescriptor) (LayerContents, error)
}

type LayerContents interface {
	Open() (io.ReadCloser, error)
}

type ImageOrImageIndexDescriptor struct {
	ImageIndex *ImageIndexDescriptor
	Image      *ImageDescriptor
}

type ImageIndexDescriptor struct {
	Refs    []string
	Images  []ImageDescriptor
	Indexes []ImageIndexDescriptor

	MediaType string
	Digest    string
	Raw       string
	Tag       string
}

type ImageDescriptor struct {
	Refs   []string
	Layers []ImageLayerDescriptor

	Config   ConfigDescriptor
	Manifest ManifestDescriptor
	Tag      string
}

type ImageLayerDescriptor struct {
	MediaType string
	Digest    string
	DiffID    string
	Size      int64
}

type ConfigDescriptor struct {
	Digest string
	Raw    string
}

type ManifestDescriptor struct {
	MediaType string
	Digest    string
	Raw       string
}

func (td ImageLayerDescriptor) IsDistributable() bool {
	// Example layer representation for windows rootfs:
	//   "mediaType": "application/vnd.docker.image.rootfs.foreign.diff.tar.gzip",
	//   "size": 1654613376,
	//   "digest": "sha256:31f9df80631e7b5d379647ee7701ff50e009bd2c03b30a67a0a8e7bba4a26f94",
	//   "urls": ["https://mcr.microsoft.com/v2/windows/servercore/blobs/sha256:31f9df80631e7b5d379647ee7701ff50e009bd2c03b30a67a0a8e7bba4a26f94"]
	return regv1types.MediaType(td.MediaType).IsDistributable()
}

func (td ImageOrImageIndexDescriptor) SortKey() string {
	switch {
	case td.ImageIndex != nil:
		return td.ImageIndex.SortKey()
	case td.Image != nil:
		return td.Image.SortKey()
	default:
		panic("ImageOrImageIndexDescriptor: expected imageIndex or image to be non-nil")
	}
}

func (td ImageIndexDescriptor) SortKey() string { return td.Digest }
func (td ImageDescriptor) SortKey() string      { return td.Manifest.Digest + "/" + td.Config.Digest }

func (t ImageOrIndex) Digest() (regv1.Hash, error) {
	switch {
	case t.Image != nil:
		return (*t.Image).Digest()
	case t.Index != nil:
		return (*t.Index).Digest()
	default:
		panic("Unknown item")
	}
}

func (t ImageOrIndex) Ref() string {
	switch {
	case t.Image != nil:
		return (*t.Image).Ref()
	case t.Index != nil:
		return (*t.Index).Ref()
	default:
		panic("Unknown item")
	}
}

func (t ImageOrIndex) Tag() string {
	switch {
	case t.Image != nil:
		return (*t.Image).Tag()
	case t.Index != nil:
		return (*t.Index).Tag()
	default:
		panic("Unknown item")
	}
}

func (t ImageOrIndex) MountableImage(registry RegistryRemoteImage) (regv1.Image, error) {
	if t.Image == nil {
		return nil, fmt.Errorf("Unable to retrieve a mountable image on an imageindex")
	}

	reference, err := regname.ParseReference((*t.Image).Ref())
	if err != nil {
		return nil, err
	}
	descriptor, err := registry.Get(reference)
	if err != nil {
		return nil, fmt.Errorf("Getting mountable image failed: %s: %s", reference, err)
	}
	imageToWrite, err := descriptor.Image()
	if err != nil {
		return nil, fmt.Errorf("Getting mountable image from descriptor failed: %s: %s", reference, err)
	}
	return imageToWrite, nil
}
