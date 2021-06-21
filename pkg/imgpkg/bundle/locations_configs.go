// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
)

const (
	locationsTagFmt string = "%s-%s.image-locations.imgpkg"
)

type LocationsNotFound struct {
	image string
}

var imageNotFoundStatusCode = map[int]struct{}{
	http.StatusNotFound:     {},
	http.StatusUnauthorized: {},
	http.StatusForbidden:    {},
}

func (n LocationsNotFound) Error() string {
	return fmt.Sprintf("Locations image in %s could not be found", n.image)
}

type LocationsConfigs struct {
	reader LocationImageReader
	logger util.LoggerWithLevels
}

type LocationImageReader interface {
	Read(img regv1.Image) (ImageLocationsConfig, error)
}

func NewLocations(logger util.LoggerWithLevels) *LocationsConfigs {
	return NewLocationsWithReader(&locationsSingleLayerReader{}, logger)
}

func NewLocationsWithReader(reader LocationImageReader, logger util.LoggerWithLevels) *LocationsConfigs {
	return &LocationsConfigs{reader: reader, logger: logger}
}

func (r LocationsConfigs) Fetch(registry image.ImagesMetadata, bundleRef name.Digest) (ImageLocationsConfig, error) {
	r.logger.Tracef("fetching Locations OCI Images for bundle: %s\n", bundleRef)
	locRef, err := r.locationsRefFromBundleRef(bundleRef)
	if err != nil {
		return ImageLocationsConfig{}, fmt.Errorf("calculating locations image tag: %s", err)
	}

	img, err := registry.Image(locRef)
	if err != nil {
		if terr, ok := err.(*transport.Error); ok {
			if _, ok := imageNotFoundStatusCode[terr.StatusCode]; ok {
				r.logger.Debugf("did not find Locations OCI Image for bundle: %s\n", bundleRef)
				return ImageLocationsConfig{}, &LocationsNotFound{image: locRef.Name()}
			}
		}
		return ImageLocationsConfig{}, fmt.Errorf("fetching location image: %s", err)
	}

	r.logger.Tracef("reading the Locations configuration file\n")
	cfg, err := r.reader.Read(img)
	if err != nil {
		return ImageLocationsConfig{}, fmt.Errorf("reading fetched location image: %s", err)
	}

	return cfg, err
}

func (r LocationsConfigs) Save(reg ImagesMetadataWriter, bundleRef name.Digest, config ImageLocationsConfig, ui ui.UI) error {
	r.logger.Tracef("saving Locations OCI Image for bundle: %s\n", bundleRef.Name())

	locRef, err := r.locationsRefFromBundleRef(bundleRef)
	if err != nil {
		return fmt.Errorf("calculating locations image tag: %s", err)
	}

	tmpDir, err := os.MkdirTemp("", "imgpkg-bundle-locations")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	err = config.WriteToPath(filepath.Join(tmpDir, LocationFilepath))
	if err != nil {
		return err
	}

	r.logger.Tracef("pushing image\n")
	_, err = plainimage.NewContents([]string{tmpDir}, nil).Push(locRef, nil, reg, ui)
	if err != nil {
		return fmt.Errorf("pushing locations image to '%s': %s", locRef.Name(), err)
	}

	return nil
}

func (r LocationsConfigs) locationsRefFromBundleRef(bundleRef name.Digest) (name.Tag, error) {
	hash, err := regv1.NewHash(bundleRef.DigestStr())
	if err != nil {
		return name.Tag{}, err
	}

	tag, err := name.NewTag(bundleRef.Context().Name())
	if err != nil {
		return name.Tag{}, err
	}

	return tag.Tag(fmt.Sprintf(locationsTagFmt, hash.Algorithm, hash.Hex)), nil
}

type locationsSingleLayerReader struct{}

func (o *locationsSingleLayerReader) Read(img regv1.Image) (ImageLocationsConfig, error) {
	conf := ImageLocationsConfig{}

	layers, err := img.Layers()
	if err != nil {
		return conf, err
	}

	if len(layers) != 1 {
		return conf, fmt.Errorf("Expected locations OCI Image to only have a single layer, got %d", len(layers))
	}

	layer := layers[0]

	mediaType, err := layer.MediaType()
	if err != nil {
		return conf, err
	}

	if mediaType != types.DockerLayer {
		return conf, fmt.Errorf("Expected layer to have docker layer media type, was %s", mediaType)
	}

	// here we know layer is .tgz so decompress and read tar headers
	unzippedReader, err := layer.Uncompressed()
	if err != nil {
		return conf, fmt.Errorf("Could not read locations image layer contents: %v", err)
	}

	tarReader := tar.NewReader(unzippedReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				return conf, fmt.Errorf("Expected to find image-locations.yml in location image")
			}
			return conf, fmt.Errorf("reading tar: %v", err)
		}

		basename := filepath.Base(header.Name)
		if basename == LocationFilepath {
			break
		}
	}

	bs, err := ioutil.ReadAll(tarReader)
	if err != nil {
		return conf, fmt.Errorf("Reading image-locations.yml from layer: %s", err)
	}

	return NewLocationConfigFromBytes(bs)
}
