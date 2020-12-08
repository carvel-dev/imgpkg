// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package lockfiles

import (
	"fmt"
	"io/ioutil"

	regname "github.com/google/go-containerregistry/pkg/name"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"gopkg.in/yaml.v2"
)

const (
	ImagesLockKind string = "ImagesLock"
	BundleLockKind string = "BundleLock"

	ImagesLockAPIVersion string = "imgpkg.carvel.dev/v1alpha1"
	BundleLockAPIVersion string = "imgpkg.carvel.dev/v1alpha1"

	BundleDir     string = ".imgpkg"
	ImageLockFile string = "images.yml"
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
		if _, err := regname.NewDigest(image.Image); err != nil {
			return fmt.Errorf("Expected ref to be in digest form, got %s", image.Image)
		}

	}

	*il = ImageLock(alias)

	return nil
}

type ImageSpec struct {
	Images []ImageDesc
}

type ImageDesc struct {
	Image       string
	Annotations map[string]string
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

func (il *ImageLock) CheckForBundles(reg ctlimg.Registry) ([]string, error) {
	var bundles []string
	for _, img := range il.Spec.Images {
		imgRef := img.Image
		parsedRef, err := regname.ParseReference(imgRef)
		if err != nil {
			return nil, err
		}
		image, err := reg.Image(parsedRef)
		if err != nil {
			return nil, err
		}

		isBundle, err := IsBundle(image)
		if err != nil {
			return nil, err
		}

		if isBundle {
			bundles = append(bundles, imgRef)
		}
	}
	return bundles, nil
}
