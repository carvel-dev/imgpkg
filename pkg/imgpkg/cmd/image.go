// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
)

type Images []Image

type Image struct {
	URL      string
	metasRaw []interface{} // populated when deserialized
}

func (imgs Images) ForImage(url string) (Image, bool) {
	for _, img := range imgs {
		if img.URL == url {
			return img, true
		}
	}
	return Image{}, false
}

// TODO only works after deserialization
func (i Image) Description() string {
	yamlBytes, err := yaml.Marshal(i.metasRaw)
	if err != nil {
		return "[]" // TODO deal better?
	}

	return strings.TrimSpace(string(yamlBytes))
}

type imageStruct struct {
	URL   string
	Metas []interface{}
}

func newImageStructs(images []Image) []imageStruct {
	var result []imageStruct
	for _, img := range images {
		result = append(result, newImageStruct(img))
	}
	return result
}

func newImageStruct(image Image) imageStruct {
	result := imageStruct{URL: image.URL}
	return result
}

func newImages(structs []imageStruct) []Image {
	var result []Image
	for _, st := range structs {
		result = append(result, Image{URL: st.URL, metasRaw: st.Metas})
	}
	return result
}

func ImageWithRepository(img string, repo string) (string, error) {
	parts := strings.Split(img, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("Parsing image URL: %s", img)
	}
	digest := parts[1]

	newURL := repo + "@" + digest
	return newURL, nil
}
