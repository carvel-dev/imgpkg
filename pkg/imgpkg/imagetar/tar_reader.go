// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	"io/ioutil"

	"github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"
)

type TarReader struct {
	path string
}

func NewTarReader(path string) TarReader {
	return TarReader{path}
}

func (r TarReader) FindByLabelKey(key string) ([]*imagedesc.ImageDescriptor, error) {
	var foundImageDescriptors []*imagedesc.ImageDescriptor

	file := tarFile{r.path}

	ids, err := r.getIdsFromManifest(file)
	if err != nil {
		return nil, err
	}

	for _, descriptor := range ids.Descriptors() {
		if descriptor.Image != nil {
			if _, found := descriptor.Image.Labels[key]; found {
				foundImageDescriptors = append(foundImageDescriptors, descriptor.Image)
			}
		}
	}
	return foundImageDescriptors, nil
}

func (r TarReader) Read() ([]imagedesc.ImageOrIndex, error) {
	file := tarFile{r.path}

	ids, err := r.getIdsFromManifest(file)
	if err != nil {
		return nil, err
	}

	return imagedesc.NewDescribedReader(ids, file).Read(), nil
}

func (r TarReader) getIdsFromManifest(file tarFile) (*imagedesc.ImageRefDescriptors, error) {
	manifestFile, err := file.Chunk("manifest.json").Open()
	if err != nil {
		return nil, err
	}

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
