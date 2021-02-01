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
	"regexp"
	"strings"
	"testing"
)

var subject CopyRepoSrc
var stdOut *bytes.Buffer

func TestMain(m *testing.M) {
	stdOut = bytes.NewBufferString("")
	logger := image.NewLogger(stdOut).NewPrefixedWriter("test|    ")
	imageSet := imageset.NewImageSet(1, logger)

	subject = CopyRepoSrc{
		logger:      logger,
		imageSet:    imageSet,
		tarImageSet: imageset.NewTarImageSet(imageSet, 1, logger),
	}

	m.Run()
}

func TestCopyingToTarBundleContainingOnlyDistributableLayers(t *testing.T) {
	bundleName := "index.docker.io/library/bundle"
	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle").WithEveryImageFrom("test_assets/image_with_config")
	defer fakeRegistry.CleanUp()

	subject := subject
	subject.BundleFlags = BundleFlags{
		bundleName,
	}
	subject.registry = fakeRegistry.Build()

	t.Run("Tar should contain every layer", func(t *testing.T) {
		bundleTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(bundleTarPath)

		err := subject.CopyToTar(bundleTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		assertTarballContainsEveryLayer(bundleTarPath, t)
	})
}

func TestCopyingToTarBundleContainingNonDistributableLayers(t *testing.T) {
	bundleName := "index.docker.io/library/bundle"
	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle_with_mult_images").
		WithEveryImageFrom("test_assets/image_with_config").
		WithNonDistributableLayerInImage("index.docker.io/library/image_with_non_distributable_layer")
	defer fakeRegistry.CleanUp()

	subject := subject
	subject.BundleFlags = BundleFlags{
		bundleName,
	}
	subject.registry = fakeRegistry.Build()

	t.Run("Tar should contain every distributable layer only", func(t *testing.T) {
		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		assertTarballContainsOnlyDistributableLayers(imageTarPath, t)
	})
	t.Run("Warning message should be printed indicating layers have been skipped", func(t *testing.T) {
		stdOut.Reset()

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		if !regexp.MustCompile("Skipped layer due to it being non-distributable\\. If you would like to include non-distributable layers, use the --include-non-distributable flag").Match(stdOut.Bytes()) {
			t.Fatalf("Expected command to give warning message, but got: %s", stdOut.String())
		}
	})

	t.Run("When Include-non-distributable flag is provided the tarball should contain every layer", func(t *testing.T) {
		subject := subject
		subject.IncludeNonDistributableFlag = IncludeNonDistributableFlag{
			IncludeNonDistributable: true,
		}

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		assertTarballContainsEveryLayer(imageTarPath, t)
	})
	t.Run("When Include-non-distributable flag is provided a warning message should not be printed", func(t *testing.T) {
		stdOut.Reset()
		subject := subject
		subject.IncludeNonDistributableFlag = IncludeNonDistributableFlag{
			IncludeNonDistributable: true,
		}

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		if strings.Contains(stdOut.String(), "Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.") {
			t.Fatalf("Expected command to not give warning message, but got: %s", stdOut.String())
		}

		if strings.Contains(stdOut.String(), "Skipped layer due to it being non-distributable.") {
			t.Fatalf("Expected command to not give warning message, but got: %s", stdOut.String())
		}
	})
}

func TestCopyingToTarImageContainingOnlyDistributableLayers(t *testing.T) {
	imageName := "index.docker.io/library/image"
	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config")
	defer fakeRegistry.CleanUp()

	subject := subject
	subject.ImageFlags = ImageFlags{
		imageName,
	}
	subject.registry = fakeRegistry.Build()

	t.Run("Tar should contain every layer", func(t *testing.T) {
		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		assertTarballContainsEveryLayer(imageTarPath, t)
	})
	t.Run("When Include-non-distributable flag is provided the tarball should contain every layer", func(t *testing.T) {
		subject := subject
		subject.IncludeNonDistributableFlag = IncludeNonDistributableFlag{
			IncludeNonDistributable: true,
		}

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		assertTarballContainsEveryLayer(imageTarPath, t)
	})
	t.Run("When Include-non-distributable flag is provided a warning message should be printed", func(t *testing.T) {
		stdOut.Reset()
		subject := subject
		subject.IncludeNonDistributableFlag = IncludeNonDistributableFlag{
			IncludeNonDistributable: true,
		}

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		if !strings.HasSuffix(stdOut.String(), "Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.\n") {
			t.Fatalf("Expected command to give warning message, but got: %s", stdOut.String())
		}
	})
}

func TestCopyingToTarImageContainingNonDistributableLayers(t *testing.T) {
	imageName := "index.docker.io/library/image"
	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config").WithNonDistributableLayer()
	defer fakeRegistry.CleanUp()
	subject := subject
	subject.ImageFlags = ImageFlags{
		imageName,
	}
	subject.registry = fakeRegistry.Build()

	t.Run("Tar should contain every distributable layer only", func(t *testing.T) {
		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		assertTarballContainsOnlyDistributableLayers(imageTarPath, t)
	})
	t.Run("When Include-non-distributable flag is provided the tarball should contain every layer", func(t *testing.T) {
		subject := subject
		subject.IncludeNonDistributableFlag = IncludeNonDistributableFlag{
			IncludeNonDistributable: true,
		}

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		assertTarballContainsEveryLayer(imageTarPath, t)
	})
}

func TestCopyingToRepoImageContainingOnlyDistributableLayers(t *testing.T) {
	imageName := "index.docker.io/library/image"
	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config")
	defer fakeRegistry.CleanUp()
	subject := subject
	subject.ImageFlags = ImageFlags{
		imageName,
	}
	subject.registry = fakeRegistry.Build()

	t.Run("When Include-non-distributable flag is provided a warning message should be printed", func(t *testing.T) {
		stdOut.Reset()
		subject := subject
		subject.IncludeNonDistributableFlag = IncludeNonDistributableFlag{
			IncludeNonDistributable: true,
		}

		_, err := subject.CopyToRepo("fakeregistry/some-repo")
		if err != nil {
			t.Fatalf("Expected CopyToRepo() to succeed but got: %s", err)
		}

		if !strings.HasSuffix(stdOut.String(), "Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.\n") {
			t.Fatalf("Expected command to give warning message, but got: %s", stdOut.String())
		}
	})
}

func assertTarballContainsEveryLayer(imageTarPath string, t *testing.T) {
	path := imagetar.NewTarReader(imageTarPath)
	imageOrIndex, err := path.Read()
	if err != nil {
		t.Fatalf("Expected to read the image tar: %s", err)
	}

	for _, imageInManifest := range imageOrIndex {
		layers, err := (*imageInManifest.Image).Layers()
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

}

func assertTarballContainsOnlyDistributableLayers(imageTarPath string, t *testing.T) {
	path := imagetar.NewTarReader(imageTarPath)
	imageOrIndex, err := path.Read()
	if err != nil {
		t.Fatalf("Expected to read the image tar: %s", err)
	}

	for _, imageInManifest := range imageOrIndex {
		layers, err := (*imageInManifest.Image).Layers()
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
