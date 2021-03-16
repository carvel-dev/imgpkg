// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
)

func TestPullBundlesWritingContentsToDisk(t *testing.T) {
	fakeUI := &bundlefakes.FakeUI{}
	pullNestedBundles := false

	t.Run("bundle referencing an image", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		fakeRegistry.WithBundleFromPath("repo/some-bundle-name", "test_assets/bundle").WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})
		subject := bundle.NewBundle(fakeRegistry.ReferenceOnTestServer("repo/some-bundle-name"), fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
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

	t.Run("bundle referencing another bundle that references another bundle does *not* pull nested bundles", func(t *testing.T) {
		// setup
		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle - dependsOn - apples/bundle
		applesBundle := fakeRegistry.WithBundleFromPath("apples/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})
		icecreamBundle := fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFromPath("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFromPath("test_assets/bundle_apples_with_single_bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle(fakeRegistry.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle"), fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		// test subject
		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
		assert.NoError(t, err)
		assert.DirExists(t, outputPath)

		// assert icecream bundle was recursively pulled onto disk
		outputDirConfigFile := filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(icecreamBundle.Digest, "sha256:", "sha256-"))
		assert.NoDirExists(t, outputDirConfigFile)

		// assert apples bundle was recursively pulled onto disk
		outputDirConfigFile = filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(applesBundle.Digest, "sha256:", "sha256-"))
		assert.NoDirExists(t, outputDirConfigFile)
	})
}

func TestPullAllNestedBundlesWritingContentsToDisk(t *testing.T) {
	fakeUI := &bundlefakes.FakeUI{}
	pullNestedBundles := true

	t.Run("bundle referencing an image", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()
		fakeRegistry.WithBundleFromPath("repo/some-bundle-name", "test_assets/bundle").WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})

		subject := bundle.NewBundle(fakeRegistry.ReferenceOnTestServer("repo/some-bundle-name"), fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
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

	t.Run("bundle referencing another bundle does pull nested bundles", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle
		icecreamBundle := fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFromPath("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle(fakeRegistry.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle"), fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		outputDirConfigFile := filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(icecreamBundle.Digest, "sha256:", "sha256-"), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err := os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err := os.ReadFile("test_assets/bundle_with_mult_images/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))
	})

	t.Run("bundle referencing another bundle that references another bundle does pull nested bundles", func(t *testing.T) {
		// setup
		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle - dependsOn - apples/bundle
		applesBundle := fakeRegistry.WithBundleFromPath("apples/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})
		iceCreamBundle := fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFromPath("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFromPath("test_assets/bundle_apples_with_single_bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle(fakeRegistry.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle"), fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		// test subject
		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
		assert.NoError(t, err)

		// assert icecream bundle was recursively pulled onto disk
		assert.DirExists(t, outputPath)
		outputDirConfigFile := filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(iceCreamBundle.Digest, "sha256:", "sha256-"), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err := os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err := os.ReadFile("test_assets/bundle_apples_with_single_bundle/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))

		// assert apples bundle was recursively pulled onto disk
		outputDirConfigFile = filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(applesBundle.Digest, "sha256:", "sha256-"), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err = os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err = os.ReadFile("test_assets/bundle_with_mult_images/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))
	})
}

func TestPullBundlesOutputToUser(t *testing.T) {
	pullNestedBundles := false

	t.Run("bundle referencing an image", func(t *testing.T) {
		output := bytes.NewBufferString("")
		writerUI := ui.NewWriterUI(output, output, nil)

		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		fakeRegistry.WithBundleFromPath("repo/some-bundle-name", "test_assets/bundle").WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})
		bundleName := fakeRegistry.ReferenceOnTestServer("repo/some-bundle-name")

		subject := bundle.NewBundle(bundleName, fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, writerUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.Regexp(t,
			fmt.Sprintf(`Pulling bundle '%s@sha256:.*'
  Extracting layer 'sha256:.*' \(1/1\)

Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update`, bundleName), output.String())
	})

	t.Run("bundle referencing another bundle", func(t *testing.T) {
		output := bytes.NewBufferString("")
		writerUI := ui.NewWriterUI(output, output, nil)
		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle
		fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFromPath("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		bundleName := fakeRegistry.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle")

		subject := bundle.NewBundle(bundleName, fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, writerUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.Regexp(t,
			fmt.Sprintf(`Pulling bundle '%s@sha256:.*'
  Extracting layer 'sha256:.*' \(1/1\)

Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update`, bundleName), output.String())
	})
}

func TestPullAllNestedBundlesOutputToUser(t *testing.T) {
	pullNestedBundles := true

	t.Run("bundle referencing another bundle", func(t *testing.T) {
		output := bytes.NewBufferString("")
		writerUI := ui.NewWriterUI(output, output, nil)
		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle
		fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFromPath("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		bundleName := fakeRegistry.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle")

		subject := bundle.NewBundle(bundleName, fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, writerUI, pullNestedBundles)
		assert.NoError(t, err)

		icecreamBundleName := fakeRegistry.ReferenceOnTestServer("icecream/bundle")
		assert.Regexp(t,
			fmt.Sprintf(`Pulling bundle '%s@sha256:.*'
  Extracting layer 'sha256:.*' \(1/1\)

Nested bundles
  Pulling nested bundle '%s@sha256:.*'
    Extracting layer 'sha256:.*' \(1/1\)

Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update`, bundleName, icecreamBundleName), output.String())
	})

	t.Run("bundle referencing multiple of the same bundles", func(t *testing.T) {
		output := bytes.NewBufferString("")
		writerUI := ui.NewWriterUI(output, output, nil)

		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_with_multiple_bundle - dependsOn - [library/image_with_a_smile, library/image_with_non_distributable_layer, library/image_with_config] - dependsOn - apples/bundle
		applesBundle := fakeRegistry.WithBundleFromPath("apples/bundle", "test_assets/bundle").WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})

		imageWithFrown := fakeRegistry.WithBundleFromPath("library/image_with_a_frown", "test_assets/bundle_apples_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: applesBundle.RefDigest},
		})
		ImageWithNonDistLayer := fakeRegistry.WithBundleFromPath("library/image_with_non_distributable_layer", "test_assets/bundle_apples_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: applesBundle.RefDigest},
		})
		imageWithSmile := fakeRegistry.WithImageFromPath("library/image_with_a_smile", "test_assets/image_with_config", map[string]string{})

		bundleWithMultipleBundles := fakeRegistry.WithBundleFromPath("repo/bundle_with_multiple_bundle", "test_assets/bundle_with_mult_images").WithImageRefs([]lockconfig.ImageRef{
			{Image: imageWithSmile.RefDigest},
			{Image: ImageWithNonDistLayer.RefDigest},
			{Image: imageWithFrown.RefDigest},
		})

		subject := bundle.NewBundle(fakeRegistry.ReferenceOnTestServer("repo/bundle_with_multiple_bundle"), fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, writerUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)

		assert.Regexp(t,
			fmt.Sprintf(`Pulling bundle '%s'
  Extracting layer 'sha256:.*' \(1/1\)

Nested bundles
  Pulling nested bundle '%s'
    Extracting layer 'sha256:.*' \(1/1\)
    Pulling nested bundle '%s'
      Extracting layer 'sha256:.*' \(1/1\)
  Pulling nested bundle '%s'
    Extracting layer 'sha256:.*' \(1/1\)
    Pulling nested bundle '%s'
    Skipped, already downloaded

Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update`, bundleWithMultipleBundles.RefDigest,
				ImageWithNonDistLayer.RefDigest,
				applesBundle.RefDigest,
				imageWithFrown.RefDigest,
				applesBundle.RefDigest), output.String())
	})

	t.Run("bundle referencing another bundle that references another bundle", func(t *testing.T) {
		// setup
		output := bytes.NewBufferString("")
		writerUI := ui.NewWriterUI(output, output, nil)

		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle - dependsOn - apples/bundle
		applesBundle := fakeRegistry.WithBundleFromPath("apples/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})
		icecreamBundle := fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFromPath("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		icecreamWithSingleBundle := fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFromPath("test_assets/bundle_apples_with_single_bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle(fakeRegistry.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle"), fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		// test subject
		err = subject.Pull(outputPath, writerUI, pullNestedBundles)
		assert.NoError(t, err)

		//assert log message
		assert.Regexp(t,
			fmt.Sprintf(`Pulling bundle '%s'
  Extracting layer 'sha256:.*' \(1/1\)

Nested bundles
  Pulling nested bundle '%s'
    Extracting layer 'sha256:.*' \(1/1\)
    Pulling nested bundle '%s'
      Extracting layer 'sha256:.*' \(1/1\)

Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update`, icecreamWithSingleBundle.RefDigest, icecreamBundle.RefDigest, applesBundle.RefDigest), output.String())
	})
}
