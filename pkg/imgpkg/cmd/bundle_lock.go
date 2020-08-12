// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

type BundleLock struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       BundleSpec
}

type BundleSpec struct {
	Image BundleImage
}

type BundleImage struct {
	Url string `yaml:"url,omitempty"`
	Tag string `yaml:"tag,omitempty"`
}
