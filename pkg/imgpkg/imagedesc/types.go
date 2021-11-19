// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagedesc

import (
	"encoding/json"
	"io"
	"sort"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	regv1types "github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/opencontainers/go-digest"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type RegistryRemoteImage interface {
	Get(reference regname.Reference) (*remote.Descriptor, error)
}

type ImageOrIndex struct {
	Image *ImageWithRef
	Index *ImageIndexWithRef

	Labels map[string]string
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

type ImageOrImageIndexDescriptors []ImageOrImageIndexDescriptor

type ImageOrImageIndexDescriptor struct {
	ImageIndex *ImageIndexDescriptor
	Image      *ImageDescriptor
}

type OCIBundle struct {
	// SchemaVersion is the image manifest schema that this image follows
	SchemaVersion int `json:"schemaVersion"`

	Manifests []ociv1.Descriptor `json:"manifests"`
}

func (i ImageOrImageIndexDescriptors) IndexFileAsBytes() ([]byte, error) {
	bundle := OCIBundle{SchemaVersion: 2}

	imgs := i
	sort.Slice(imgs, func(i, j int) bool {
		return imgs[i].SortKey() < imgs[j].SortKey()
	})

	for _, descriptor := range imgs {
		var manifest ociv1.Manifest
		err := json.Unmarshal([]byte(descriptor.Image.Manifest.Raw), &manifest)
		if err != nil {
			return nil, err

		}

		imgDescriptor := ociv1.Descriptor{
			MediaType: "application/vnd.oci.image.manifest.v1+json",
			Digest:    descriptor.ManifestDigest(),
			Size:      descriptor.ManifestSize(),
			Annotations: map[string]string{
				ociv1.AnnotationRefName: descriptor.Image.Tag,
			},
		}

		bundle.Manifests = append(bundle.Manifests, imgDescriptor)
	}

	return json.Marshal(bundle)
}

func (i ImageOrImageIndexDescriptor) ManifestDigest() digest.Digest {
	if i.Image != nil {
		return digest.Digest(i.Image.Manifest.Digest)
	}

	return digest.Digest(i.ImageIndex.Digest)
}

func (i ImageOrImageIndexDescriptor) ManifestAsBytes() ([]byte, error) {
	if i.Image != nil {
		return []byte(i.Image.Manifest.Raw), nil
	}

	return []byte(i.ImageIndex.Raw), nil
}

func (i ImageOrImageIndexDescriptor) ConfigDigest() digest.Digest {
	if i.Image != nil {
		return digest.Digest(i.Image.Config.Digest)
	}

	return ""
}

func (i ImageOrImageIndexDescriptor) ConfigAsBytes() ([]byte, error) {
	if i.Image != nil {
		return []byte(i.Image.Config.Raw), nil
	}

	return nil, nil
}

func (i ImageOrImageIndexDescriptor) ManifestSize() int64 {
	if i.Image != nil {
		return int64(len(i.Image.Manifest.Raw))
	}

	return int64(len(i.ImageIndex.Raw))
}

type ImageIndexDescriptor struct {
	Refs    []string
	Images  []ImageDescriptor
	Indexes []ImageIndexDescriptor

	MediaType string
	Digest    string
	Raw       string
	Tag       string

	Labels map[string]string
}

type ImageDescriptor struct {
	Refs   []string
	Layers []ImageLayerDescriptor

	Config   ConfigDescriptor
	Manifest ManifestDescriptor
	Tag      string

	Labels map[string]string
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
	ociv1.Manifest
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
