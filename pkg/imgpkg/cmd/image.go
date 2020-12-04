// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"
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

func ImageWithRepository(img string, repo string) (string, error) {
	parts := strings.Split(img, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("Parsing image URL: %s", img)
	}
	digest := parts[1]

	newURL := repo + "@" + digest
	return newURL, nil
}
