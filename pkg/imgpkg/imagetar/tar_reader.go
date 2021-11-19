// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"

	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagedesc"
)

type TarReader struct {
	path string
}

func NewTarReader(path string) TarReader {
	return TarReader{path}
}

func (r TarReader) Read() ([]imagedesc.ImageOrIndex, error) {
	file := tarFile{r.path}

	var ids *imagedesc.ImageRefDescriptors
	var err error
	if r.isOCICompliant(file) {
		ids, err = r.getIdsFromOCIImage(file)
		if err != nil {
			return nil, err
		}
	} else {
		ids, err = r.getIdsFromManifest(file)
		if err != nil {
			return nil, err
		}
	}

	return imagedesc.NewDescribedReader(ids, file).Read(), nil
}

func (r TarReader) isOCICompliant(file tarFile) bool {
	if ok, _ := file.existsChunk("oci-layout"); !ok {
		return false
	}
	return true
}

func (r TarReader) getIdsFromManifest(file tarFile) (*imagedesc.ImageRefDescriptors, error) {
	manifestFile, err := file.Chunk("manifest.json").Open()
	if err != nil {
		return nil, err
	}
	defer manifestFile.Close()

	manifestBytes, err := ioutil.ReadAll(manifestFile)
	if err != nil {
		return nil, err
	}

	ids, err := imagedesc.NewImageRefDescriptorsFromBytes(manifestBytes)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r TarReader) getIdsFromOCIImage(file tarFile) (*imagedesc.ImageRefDescriptors, error) {
	indexFile, err := file.Chunk("index.json").Open()
	if err != nil {
		return nil, err
	}
	defer indexFile.Close()

	indexBytes, err := ioutil.ReadAll(indexFile)
	if err != nil {
		return nil, err
	}

	var bundle imagedesc.OCIBundle
	err = json.Unmarshal(indexBytes, &bundle)
	if err != nil {
		return nil, err
	}

	var manifests []ociv1.Manifest
	for _, idxManifest := range bundle.Manifests {
		digestParts := strings.Split(string(idxManifest.Digest), ":")
		manifestFile, err := file.Chunk(filepath.Join("blobs", digestParts[0], digestParts[1])).Open()
		manifestBytes, err := ioutil.ReadAll(manifestFile)
		if err != nil {
			return nil, err
		}
		err = manifestFile.Close()
		if err != nil {
			return nil, err
		}

		var manifest imagedesc.ManifestDescriptor
		err = json.Unmarshal(manifestBytes, &manifest)
		if err != nil {
			return nil, err
		}

		digestParts = strings.Split(string(manifest.Config.Digest), ":")
		configFile, err := file.Chunk(filepath.Join("blobs", digestParts[0], digestParts[1])).Open()
		configBytes, err := ioutil.ReadAll(configFile)
		if err != nil {
			return nil, err
		}
		err = configFile.Close()
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(configBytes, &manifest.Config)
		if err != nil {
			return nil, err
		}
	}

	ids, err := imagedesc.NewImageRefDescriptorsFromOCIManifests(manifests)
}
