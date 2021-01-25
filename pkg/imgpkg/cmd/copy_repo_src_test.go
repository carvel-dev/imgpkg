// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"archive/tar"
	"bytes"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyRepoToTar(t *testing.T) {
	bundleName := "index.docker.io/library/bundle"

	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle")
	fakeRegistry.WithImageFromPath("index.docker.io/library/image_with_config", "test_assets/image_with_config")
	defer fakeRegistry.CleanUp()

	logger := image.NewLogger(os.Stdout).NewPrefixedWriter("test-imageset")
	imageSet := imageset.NewImageSet(1, logger)
	src := CopyRepoSrc{
		logger: logger,
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

	logger := image.NewLogger(os.Stdout).NewPrefixedWriter("test-imageset")
	imageSet := imageset.NewImageSet(1, logger)
	src := CopyRepoSrc{
		logger: logger,
		IncludeNonDistributableFlag: IncludeNonDistributableFlag{
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

	for _, layer := range layers {
		digest, err := layer.Digest()
		if err != nil {
			t.Fatalf("Expected generating a digest from a layer to succeed got: %s", err)
		}

		if !doesLayerExistInTarball(imageTarPath, digest, t) {
			t.Fatalf("Expected to find layer [%s] in tarball, but did not", digest)
		}
	}
}

func TestCopyRepoToTarWithoutNonDistributableFlagButImageHasANonDistributableLayer(t *testing.T) {
	imageName := "index.docker.io/library/image"

	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config").WithNonDistributableLayer()

	defer fakeRegistry.CleanUp()

	logger := image.NewLogger(os.Stdout).NewPrefixedWriter("test-imageset")
	imageSet := imageset.NewImageSet(1, logger)
	src := CopyRepoSrc{
		logger: logger,
		IncludeNonDistributableFlag: IncludeNonDistributableFlag{
			IncludeNonDistributable: false,
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

	for _, layer := range layers {
		mediaType, err := layer.MediaType()
		if err != nil {
			t.Fatalf(err.Error())
		}

		digest, err := layer.Digest()
		if err != nil {
			t.Fatalf("Expected generating a digest from a layer to succeed got: %s", err)
		}

		if doesLayerExistInTarball(imageTarPath, digest, t) && !mediaType.IsDistributable() {
			t.Fatalf("Expected to fail. The foreign layer was found in the tarball when we expected it not to")
		}
	}
}

func TestCopyRepoToTarWithNonDistributableFlagButEachLayersIsCompressedLayerWarningMessage(t *testing.T) {
	imageName := "index.docker.io/library/image"

	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config")

	defer fakeRegistry.CleanUp()

	output := bytes.NewBufferString("")
	logger := image.NewLogger(output).NewPrefixedWriter("test-imageset")
	imageSet := imageset.NewImageSet(1, logger)
	src := CopyRepoSrc{
		logger: logger,
		IncludeNonDistributableFlag: IncludeNonDistributableFlag{
			IncludeNonDistributable: true,
		},
		ImageFlags: ImageFlags{
			imageName,
		},
		imageSet:    imageSet,
		tarImageSet: imageset.NewTarImageSet(imageSet, 1, logger),
		registry:    fakeRegistry.Build(),
	}

	imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
	defer os.Remove(imageTarPath)

	err := src.CopyToTar(imageTarPath)
	if err != nil {
		t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
	}

	if !strings.HasSuffix(output.String(), "Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.\n") {
		t.Fatalf("Expected command to give warning message, but got: %s", output.String())
	}
}

func TestCopyRepoToTarWithNonDistributableFlagButOneLayerIsNonDistributableNoWarningMessage(t *testing.T) {
	bundleName := "index.docker.io/library/bundle"

	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle_with_mult_images")
	fakeRegistry.WithImageFromPath("index.docker.io/library/image_with_config", "test_assets/image_with_config")
	fakeRegistry.WithImageFromPath("index.docker.io/library/image_with_non_distributable_layer", "test_assets/image_with_config").WithNonDistributableLayer()
	fakeRegistry.WithImageFromPath("index.docker.io/library/image_with_a_smile", "test_assets/image_with_config")
	defer fakeRegistry.CleanUp()

	output := bytes.NewBufferString("")
	logger := image.NewLogger(output).NewPrefixedWriter("test-imageset")
	imageSet := imageset.NewImageSet(1, logger)
	src := CopyRepoSrc{
		logger: logger,
		IncludeNonDistributableFlag: IncludeNonDistributableFlag{
			IncludeNonDistributable: true,
		},
		BundleFlags: BundleFlags{
			bundleName,
		},
		imageSet:    imageSet,
		tarImageSet: imageset.NewTarImageSet(imageSet, 1, image.NewLogger(output).NewPrefixedWriter("test-tarImageSet")),
		registry:    fakeRegistry.Build(),
	}

	imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
	defer os.Remove(imageTarPath)

	err := src.CopyToTar(imageTarPath)
	if err != nil {
		t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
	}

	if strings.Contains(output.String(), "Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.") {
		t.Fatalf("Expected command to not give warning message, but got: %s", output.String())
	}
}

func doesLayerExistInTarball(path string, digest regv1.Hash, t *testing.T) bool {
	filePathInTar := digest.Algorithm + "-" + digest.Hex + ".tar.gz"
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Expected to open tarball. Got error: %s", err)
	}
	tf := tar.NewReader(file)
	for {
		hdr, err := tf.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Expected to read next tarball entry. Got error: %s", err)
		}
		if hdr.Name == filePathInTar {
			return true
		}
	}
	return false
}
