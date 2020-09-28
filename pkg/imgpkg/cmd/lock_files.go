// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"io/ioutil"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	ImageLockKind  string = "ImageLock"
	BundleLockKind string = "BundleLock"

	ImageLockAPIVersion  string = "imgpkg.k14s.io/v1alpha1"
	BundleLockAPIVersion string = "imgpkg.k14s.io/v1alpha1"
)

type BundleLock struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       BundleSpec
}

type BundleSpec struct {
	Image ImageLocation
}

type ImageLock struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       ImageSpec
}

func (il *ImageLock) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// needed to avoid infinite recursion
	type imageLockAlias ImageLock

	var alias imageLockAlias
	err := unmarshal(&alias)
	if err != nil {
		return err
	}

	for _, image := range alias.Spec.Images {
		if _, err := name.NewDigest(image.DigestRef); err != nil {
			return errors.Errorf("Expected ref to be in digest form, got %s", image.DigestRef)
		}
	}

	*il = ImageLock(alias)

	return nil
}

type ImageSpec struct {
	Images []ImageDesc
}

type ImageDesc struct {
	ImageLocation `yaml:",inline"`
	Name          string
	Metadata      string
}

type ImageLocation struct {
	DigestRef   string `yaml:"url,omitempty"`
	OriginalTag string `yaml:"tag,omitempty"`
}

type Lock struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

func ReadLockFile(path string) (Lock, error) {
	var lock Lock
	err := readPathInto(path, &lock)

	return lock, err
}

func ReadBundleLockFile(path string) (BundleLock, error) {
	var bundleLock BundleLock
	err := readPathInto(path, &bundleLock)

	return bundleLock, err
}

func ReadImageLockFile(path string) (ImageLock, error) {
	var imgLock ImageLock
	err := readPathInto(path, &imgLock)

	return imgLock, err
}

func readPathInto(path string, obj interface{}) error {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(bs, obj)
}
