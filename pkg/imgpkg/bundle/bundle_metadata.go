// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"archive/tar"
	"fmt"
	"io"
	"path/filepath"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"sigs.k8s.io/yaml"
)

// BundleAuthor contains author information from a bundle's metadata file.
type BundleAuthor struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// BundleWebsite contains a website from a bundle's metadata file.
type BundleWebsite struct {
	URL string `json:"url,omitempty"`
}

// BundleMetadata contains the user-provided information in .imgpkg/bundle.yml.
type BundleMetadata struct {
	Metadata map[string]string `json:"metadata,omitempty"`
	Authors  []BundleAuthor    `json:"authors,omitempty"`
	Websites []BundleWebsite   `json:"websites,omitempty"`
}

// Metadata reads the metadata associated with the bundle.
func (o *Bundle) Metadata() (BundleMetadata, error) {
	img, err := o.checkedImage()
	if err != nil {
		return BundleMetadata{}, err
	}

	return readBundleMetadata(img)
}

func readBundleMetadata(img regv1.Image) (BundleMetadata, error) {
	metadata := BundleMetadata{}
	layers, err := img.Layers()
	if err != nil {
		return metadata, err
	}

	if len(layers) != 1 {
		return metadata, fmt.Errorf("Expected bundle to only have a single layer, got %d", len(layers))
	}

	layer := layers[0]
	mediaType, err := layer.MediaType()
	if err != nil {
		return metadata, err
	}

	if mediaType != types.DockerLayer {
		return metadata, fmt.Errorf("Expected layer to have docker layer media type, was %s", mediaType)
	}

	uncompressedReader, err := layer.Uncompressed()
	if err != nil {
		return metadata, fmt.Errorf("Could not read bundle image layer contents: %v", err)
	}
	defer uncompressedReader.Close()

	tarReader := tar.NewReader(uncompressedReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				return metadata, nil
			}
			return metadata, fmt.Errorf("reading tar: %v", err)
		}

		if filepath.Dir(header.Name) == ImgpkgDir && filepath.Base(header.Name) == BundleMetadataFile {
			break
		}
	}

	contents, err := io.ReadAll(tarReader)
	if err != nil {
		return metadata, fmt.Errorf("Reading bundle.yml from layer: %s", err)
	}

	if err := yaml.Unmarshal(contents, &metadata); err != nil {
		return metadata, fmt.Errorf("Unmarshalling bundle metadata: %s", err)
	}

	return metadata, nil
}
