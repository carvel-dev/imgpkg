// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyRepoToTar(t *testing.T) {
	bundleName := "index.docker.io/library/bundle"

	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle")
	fakeRegistry.WithImageFromPath("index.docker.io/library/image_with_config", "test_assets/image_with_config")
	defer fakeRegistry.CleanUp()

	imageSet := imageset.NewImageSet(1, image.NewLogger(os.Stdout).NewPrefixedWriter("test-imageset"))
	src := CopyRepoSrc{
		BundleFlags: BundleFlags{
			bundleName,
		},
		imageSet:    imageSet,
		tarImageSet: imageset.NewTarImageSet(imageSet, 1, image.NewLogger(os.Stdout).NewPrefixedWriter("test-tarImageSet")),
		registry:    fakeRegistry.Build(),
	}

	bundleTarPath := filepath.Join(os.TempDir(), "bundle.tar")
	defer os.Remove(bundleTarPath)

	err := src.CopyToTar(bundleTarPath)
	if err != nil {
		t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
	}

	bundleFileInfo, err := os.Stat(bundleTarPath)
	if err == os.ErrNotExist {
		t.Fatalf("Bundle tar file not found: %s", err)
	}
	if err != nil {
		t.Fatalf("Getting bundle tar: %s", err)
	}
	if bundleFileInfo.Size() <= 0 {
		t.Fatalf("Expected bundle tar to have size > 0, but was empty")
	}
}
