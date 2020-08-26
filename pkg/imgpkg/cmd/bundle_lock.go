// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
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
