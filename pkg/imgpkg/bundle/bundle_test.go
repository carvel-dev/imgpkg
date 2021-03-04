// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/stretchr/testify/assert"
)

func TestPullBundlesRecursivelyWritingContentsToDisk(t *testing.T) {
	fakeUI := &bundlefakes.FakeUI{}
	recursiveBundles := true

	t.Run("bundle referencing an image", func(t *testing.T) {
		fakeRegistry := NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		fakeRegistry.WithBundleFromPath("repo/some-bundle-name", "test_assets/bundle").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		subject := bundle.NewBundle("repo/some-bundle-name", fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, recursiveBundles)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		outputDirConfigFile := filepath.Join(outputPath, "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err := os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err := os.ReadFile("test_assets/bundle/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))
	})

	t.Run("bundle referencing another bundle", func(t *testing.T) {
		fakeRegistry := NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle
		fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle("repo/bundle_icecream_with_single_bundle", fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, recursiveBundles)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		digest, err := fakeRegistry.state["index.docker.io/icecream/bundle:latest"].image.Digest()
		assert.NoError(t, err)

		outputDirConfigFile := filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(digest.String(), "sha256:", "sha256-"), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err := os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err := os.ReadFile("test_assets/bundle_with_mult_images/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))
	})

	t.Run("bundle referencing another bundle that references another bundle", func(t *testing.T) {
		// setup
		fakeRegistry := NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle - dependsOn - apples/bundle
		fakeRegistry.WithBundleFromPath("apples/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_apples_with_single_bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle("repo/bundle_icecream_with_single_bundle", fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		// test subject
		err = subject.Pull(outputPath, fakeUI, recursiveBundles)
		assert.NoError(t, err)

		// assert icecream bundle was recursively pulled onto disk
		assert.DirExists(t, outputPath)
		digest, err := fakeRegistry.state["index.docker.io/icecream/bundle:latest"].image.Digest()
		assert.NoError(t, err)

		outputDirConfigFile := filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(digest.String(), "sha256:", "sha256-"), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err := os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err := os.ReadFile("test_assets/bundle_apples_with_single_bundle/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))

		// assert apples bundle was recursively pulled onto disk
		digest, err = fakeRegistry.state["index.docker.io/apples/bundle:latest"].image.Digest()
		assert.NoError(t, err)

		outputDirConfigFile = filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(digest.String(), "sha256:", "sha256-"), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err = os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err = os.ReadFile("test_assets/bundle_with_mult_images/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))
	})
}

func TestPullOutputToUser(t *testing.T) {
	t.Run("bundle referencing another bundle", func(t *testing.T) {
		output := bytes.NewBufferString("")
		writerUI := ui.NewWriterUI(output, output, nil)
		fakeRegistry := NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle
		fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle("repo/bundle_icecream_with_single_bundle", fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, writerUI, true)
		assert.NoError(t, err)

		assert.Regexp(t,
			`Pulling bundle 'index.docker.io/repo/bundle_icecream_with_single_bundle@sha256:.*'
Bundle Layers
  Extracting layer 'sha256:.*' \(1/1\)
Nested bundles
  Pulling Nested bundle 'index.docker.io/icecream/bundle@sha256:.*'
    Extracting layer 'sha256:.*' \(1/1\)
Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update`, output.String())
	})

	t.Run("bundle referencing another bundle that references another bundle", func(t *testing.T) {
		// setup
		output := bytes.NewBufferString("")
		writerUI := ui.NewWriterUI(output, output, nil)

		fakeRegistry := NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle - dependsOn - apples/bundle
		fakeRegistry.WithBundleFromPath("apples/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_apples_with_single_bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle("repo/bundle_icecream_with_single_bundle", fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		// test subject
		err = subject.Pull(outputPath, writerUI, true)
		assert.NoError(t, err)

		//assert log message
		assert.Regexp(t,
			`Pulling bundle 'index.docker.io/repo/bundle_icecream_with_single_bundle@sha256:.*'
Bundle Layers
  Extracting layer 'sha256:.*' \(1/1\)
Nested bundles
  Pulling Nested bundle 'index.docker.io/icecream/bundle@sha256:.*'
    Extracting layer 'sha256:.*' \(1/1\)
    Pulling Nested bundle 'index.docker.io/apples/bundle@sha256:.*'
      Extracting layer 'sha256:.*' \(1/1\)
Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update`, output.String())
	})
}
