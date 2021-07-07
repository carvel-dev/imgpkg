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

func (r TarReader) Read() (*imagedesc.ImageDescriptor, []imagedesc.ImageOrIndex, error) {
	file := tarFile{r.path}

	manifestFile, err := file.Chunk("manifest.json").Open()
	if err != nil {
		return nil, nil, err
	}

	manifestBytes, err := ioutil.ReadAll(manifestFile)
	if err != nil {
		return nil, nil, err
	}

	ids, err := imagedesc.NewImageRefDescriptorsFromBytes(manifestBytes)
	if err != nil {
		return nil, nil, err
	}

	var mainBundle *imagedesc.ImageDescriptor
	for _, descriptor := range ids.Descriptors() {
		if descriptor.Image != nil {
			if _, found := descriptor.Image.Labels["main.bundle"]; found {
				mainBundle = descriptor.Image
			}
		}
	}

	return mainBundle, imagedesc.NewDescribedReader(ids, file).Read(), nil
}
