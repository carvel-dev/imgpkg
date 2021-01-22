// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
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

func TestCopyRepoToTarWhenNonDistributableFlagIsProvided(t *testing.T) {
	imageName := "index.docker.io/library/image"

	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config").WithNonDistributableLayer()

	defer fakeRegistry.CleanUp()

	imageSet := imageset.NewImageSet(1, image.NewLogger(os.Stdout).NewPrefixedWriter("test-imageset"))
	src := CopyRepoSrc{
		NonDistributableFlag: NonDistributableFlag{
			IncludeNonDistributable: true,
		},
		ImageFlags: ImageFlags{
			imageName,
		},
		imageSet:    imageSet,
		tarImageSet: imageset.NewTarImageSet(imageSet, 1, image.NewLogger(os.Stdout).NewPrefixedWriter("test-tarImageSet")),
		registry:    fakeRegistry.Build(),
	}

	imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
	defer os.Remove(imageTarPath)

	err := src.CopyToTar(imageTarPath)
	if err != nil {
		t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
	}

	path := imagetar.NewTarReader(imageTarPath)
	imageOrIndex, err := path.Read()
	if err != nil {
		t.Fatalf("Expected to read the image tar: %s", err)
	}

	layers, err := (*imageOrIndex[0].Image).Layers()
	if err != nil {
		t.Fatalf("Expected image tar to contain layers: %s", err)
	}

	if len(layers) != 3 {
		t.Fatalf("Expected 3 files in the image tar file, but got %d", len(layers))
	}
}
