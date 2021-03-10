// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/stretchr/testify/assert"
)

func TestPullBundlesWritingContentsToDisk(t *testing.T) {
	fakeUI := &bundlefakes.FakeUI{}
	pullNestedBundles := false

	t.Run("bundle referencing an image", func(t *testing.T) {
		fakeImagesMetadataBuilder := NewFakeImagesMetadataBuilder(t)
		defer fakeImagesMetadataBuilder.CleanUp()

		fakeImagesMetadataBuilder.WithBundleFromPath("repo/some-bundle-name", "test_assets/bundle").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		subject := bundle.NewBundle(fakeImagesMetadataBuilder.ReferenceOnTestServer("repo/some-bundle-name"), fakeImagesMetadataBuilder.Build())
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
		fakeImagesMetadataBuilder := NewFakeImagesMetadataBuilder(t)
		defer fakeImagesMetadataBuilder.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle - dependsOn - apples/bundle
		applesBundle := fakeImagesMetadataBuilder.WithBundleFromPath("apples/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		icecreamBundle := fakeImagesMetadataBuilder.WithBundleFromPath("icecream/bundle", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeImagesMetadataBuilder.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_apples_with_single_bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle(fakeImagesMetadataBuilder.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle"), fakeImagesMetadataBuilder.Build())
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
		fakeImagesMetadataBuilder := NewFakeImagesMetadataBuilder(t)
		defer fakeImagesMetadataBuilder.CleanUp()
		fakeImagesMetadataBuilder.WithBundleFromPath("repo/some-bundle-name", "test_assets/bundle").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})

		subject := bundle.NewBundle(fakeImagesMetadataBuilder.ReferenceOnTestServer("repo/some-bundle-name"), fakeImagesMetadataBuilder.Build())
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
		fakeImagesMetadataBuilder := NewFakeImagesMetadataBuilder(t)
		defer fakeImagesMetadataBuilder.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle
		icecreamBundle := fakeImagesMetadataBuilder.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeImagesMetadataBuilder.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle(fakeImagesMetadataBuilder.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle"), fakeImagesMetadataBuilder.Build())
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
		fakeImagesMetadataBuilder := NewFakeImagesMetadataBuilder(t)
		defer fakeImagesMetadataBuilder.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle - dependsOn - apples/bundle
		applesBundle := fakeImagesMetadataBuilder.WithBundleFromPath("apples/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		iceCreamBundle := fakeImagesMetadataBuilder.WithBundleFromPath("icecream/bundle", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeImagesMetadataBuilder.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_apples_with_single_bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle(fakeImagesMetadataBuilder.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle"), fakeImagesMetadataBuilder.Build())
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

		fakeImagesMetadataBuilder := NewFakeImagesMetadataBuilder(t)
		defer fakeImagesMetadataBuilder.CleanUp()

		fakeImagesMetadataBuilder.WithBundleFromPath("repo/some-bundle-name", "test_assets/bundle").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		bundleName := fakeImagesMetadataBuilder.ReferenceOnTestServer("repo/some-bundle-name")

		subject := bundle.NewBundle(bundleName, fakeImagesMetadataBuilder.Build())
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
		fakeImagesMetadataBuilder := NewFakeImagesMetadataBuilder(t)
		defer fakeImagesMetadataBuilder.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle
		fakeImagesMetadataBuilder.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeImagesMetadataBuilder.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		bundleName := fakeImagesMetadataBuilder.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle")

		subject := bundle.NewBundle(bundleName, fakeImagesMetadataBuilder.Build())
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
		fakeImagesMetadataBuilder := NewFakeImagesMetadataBuilder(t)
		defer fakeImagesMetadataBuilder.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle
		fakeImagesMetadataBuilder.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeImagesMetadataBuilder.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		bundleName := fakeImagesMetadataBuilder.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle")

		subject := bundle.NewBundle(bundleName, fakeImagesMetadataBuilder.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, writerUI, pullNestedBundles)
		assert.NoError(t, err)

		icecreamBundleName := fakeImagesMetadataBuilder.ReferenceOnTestServer("icecream/bundle")
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

		fakeImagesMetadataBuilder := NewFakeImagesMetadataBuilder(t)
		defer fakeImagesMetadataBuilder.CleanUp()

		// repo/bundle_with_multiple_bundle - dependsOn - [library/image_with_a_smile, library/image_with_non_distributable_layer, library/image_with_config] - dependsOn - apples/bundle
		fakeImagesMetadataBuilder.WithBundleFromPath("apples/bundle", "test_assets/bundle").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})

		fakeImagesMetadataBuilder.WithBundleFromPath("library/image_with_a_frown", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFrom("test_assets/bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeImagesMetadataBuilder.WithBundleFromPath("library/image_with_non_distributable_layer", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFrom("test_assets/bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeImagesMetadataBuilder.WithImageFromPath("library/image_with_a_smile", "test_assets/image_with_config", map[string]string{})

		fakeImagesMetadataBuilder.WithBundleFromPath("repo/bundle_with_multiple_bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/bundle_apples_with_single_bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle(fakeImagesMetadataBuilder.ReferenceOnTestServer("repo/bundle_with_multiple_bundle"), fakeImagesMetadataBuilder.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, writerUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)

		registryURL, err := url.Parse(fakeImagesMetadataBuilder.server.URL)
		assert.NoError(t, err)

		assert.Regexp(t,
			fmt.Sprintf(`Pulling bundle '%[1]s/repo/bundle_with_multiple_bundle@sha256:.*'
  Extracting layer 'sha256:.*' \(1/1\)

Nested bundles
  Pulling nested bundle '%[1]s/library/image_with_a_frown@sha256:.*'
    Extracting layer 'sha256:.*' \(1/1\)
    Pulling nested bundle '%[1]s/apples/bundle@sha256:.*'
      Extracting layer 'sha256:.*' \(1/1\)
  Pulling nested bundle '%[1]s/library/image_with_non_distributable_layer@sha256:.*'
    Extracting layer 'sha256:.*' \(1/1\)
    Pulling nested bundle '%[1]s/apples/bundle@sha256:.*'
    Skipped, already downloaded
  Pulling nested bundle '%[1]s/library/image_with_a_smile@sha256:.*'
    Extracting layer 'sha256:.*' \(1/1\)
    Pulling nested bundle '%[1]s/apples/bundle@sha256:.*'
    Skipped, already downloaded

Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update`, registryURL.Host), output.String())
	})

	t.Run("bundle referencing another bundle that references another bundle", func(t *testing.T) {
		// setup
		output := bytes.NewBufferString("")
		writerUI := ui.NewWriterUI(output, output, nil)

		fakeImagesMetadataBuilder := NewFakeImagesMetadataBuilder(t)
		defer fakeImagesMetadataBuilder.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle - dependsOn - apples/bundle
		fakeImagesMetadataBuilder.WithBundleFromPath("apples/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeImagesMetadataBuilder.WithBundleFromPath("icecream/bundle", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeImagesMetadataBuilder.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_apples_with_single_bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle(fakeImagesMetadataBuilder.ReferenceOnTestServer("repo/bundle_icecream_with_single_bundle"), fakeImagesMetadataBuilder.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		// test subject
		err = subject.Pull(outputPath, writerUI, pullNestedBundles)
		assert.NoError(t, err)

		//assert log message
		registryURL, err := url.Parse(fakeImagesMetadataBuilder.server.URL)
		assert.NoError(t, err)

		assert.Regexp(t,
			fmt.Sprintf(`Pulling bundle '%[1]s/repo/bundle_icecream_with_single_bundle@sha256:.*'
  Extracting layer 'sha256:.*' \(1/1\)

Nested bundles
  Pulling nested bundle '%[1]s/icecream/bundle@sha256:.*'
    Extracting layer 'sha256:.*' \(1/1\)
    Pulling nested bundle '%[1]s/apples/bundle@sha256:.*'
      Extracting layer 'sha256:.*' \(1/1\)

Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update`, registryURL.Host), output.String())
	})
}
