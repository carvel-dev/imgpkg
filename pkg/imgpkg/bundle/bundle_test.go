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
	goui "github.com/cppforlife/go-cli-ui/ui"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
)

func TestPullBundleWritingContentsToDisk(t *testing.T) {
	fakeUI := &bundlefakes.FakeUI{}
	pullNestedBundles := false

	t.Run("bundle referencing an image", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
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
		assert.Equal(t, string(expectedConfigFile), string(actualConfigFile))
	})

	t.Run("bundle referencing another bundle that references another bundle does *not* pull nested bundles", func(t *testing.T) {
		// setup
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
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

func TestPullNestedBundlesWritingContentsToDisk(t *testing.T) {
	fakeUI := &bundlefakes.FakeUI{}
	pullNestedBundles := true

	t.Run("bundle referencing an image", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
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
		assert.Equal(t, string(expectedConfigFile), string(actualConfigFile))
	})

	t.Run("bundle referencing another bundle does pull nested bundles", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
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
		assert.Equal(t, string(expectedConfigFile), string(actualConfigFile))
	})

	t.Run("bundle referencing another bundle that references another bundle does pull nested bundles", func(t *testing.T) {
		// setup
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
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
		assert.Equal(t, string(expectedConfigFile), string(actualConfigFile))

		// assert apples bundle was recursively pulled onto disk
		outputDirConfigFile = filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(applesBundle.Digest, "sha256:", "sha256-"), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err = os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err = os.ReadFile("test_assets/bundle_with_mult_images/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(expectedConfigFile), string(actualConfigFile))
	})
}

func TestPullNestedBundlesLocalizesImagesLockFile(t *testing.T) {
	fakeUI := &bundlefakes.FakeUI{}
	pullNestedBundles := true

	t.Run("bundle referencing another bundle in the same repo updates both bundle's imageslock", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()

		randomImageColocatedWithIcecreamBundle := fakeRegistry.WithRandomImage("icecream/bundle")
		randomImageFromPrivateRegistry := fakeRegistry.WithImage("library/image1", randomImageColocatedWithIcecreamBundle.Image)

		icecreamBundle := fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImageFromPrivateRegistry.RefDigest},
		})

		randomBundleCollocatedWithRootBundle := fakeRegistry.WithImage("repo/bundle-with-collocated-bundles", icecreamBundle.Image)
		randomImageCollocatedWithRootBundle := fakeRegistry.WithImage("repo/bundle-with-collocated-bundles", randomImageColocatedWithIcecreamBundle.Image)

		rootBundle := fakeRegistry.WithBundleFromPath("repo/bundle-with-collocated-bundles", "test_assets/bundle_icecream_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: icecreamBundle.RefDigest},
		})

		subject := bundle.NewBundle(rootBundle.RefDigest, fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		rootBundleImagesYmlFile := filepath.Join(outputPath, ".imgpkg", "images.yml")
		assert.FileExists(t, rootBundleImagesYmlFile)
		rootImagesYmlFile, err := os.ReadFile(rootBundleImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- image: %s
kind: ImagesLock
`, randomBundleCollocatedWithRootBundle.RefDigest), string(rootImagesYmlFile))

		outputDirImagesYmlFile := filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(icecreamBundle.Digest, "sha256:", "sha256-"), ".imgpkg", "images.yml")
		assert.FileExists(t, outputDirImagesYmlFile)
		nestedImagesYmlFile, err := os.ReadFile(outputDirImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- image: %s
kind: ImagesLock
`, randomImageCollocatedWithRootBundle.RefDigest), string(nestedImagesYmlFile))
	})

	t.Run("bundle referencing two bundles, only 1 is relocated, should update only the 1 that is relocated imageslock", func(t *testing.T) {
		fakePublicRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakePublicRegistry.CleanUp()

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()

		randomImageFromFakeRegistry := fakeRegistry.WithRandomImage("icecream/bundle")
		randomImageFromPublicRegistry := fakePublicRegistry.WithImage("library/image1", randomImageFromFakeRegistry.Image)

		icecreamBundle := fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImageFromPublicRegistry.RefDigest},
		})

		appleBundle := fakeRegistry.WithBundleFromPath("apple/bundle", "test_assets/bundle_apples_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImageFromPublicRegistry.RefDigest},
		})

		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_and_apple", "test_assets/bundle_icecream_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: icecreamBundle.RefDigest},
			{Image: appleBundle.RefDigest},
		})

		fakePublicRegistry.Build()
		subject := bundle.NewBundle(fakeRegistry.ReferenceOnTestServer("repo/bundle_icecream_and_apple"), fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
		assert.NoError(t, err)

		outputDirImagesYmlFile := filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(icecreamBundle.Digest, "sha256:", "sha256-"), ".imgpkg", "images.yml")
		assert.FileExists(t, outputDirImagesYmlFile)
		actualImagesYmlFile, err := os.ReadFile(outputDirImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- image: %s
kind: ImagesLock
`, randomImageFromFakeRegistry.RefDigest), string(actualImagesYmlFile))

		outputDirImagesYmlFile = filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(appleBundle.Digest, "sha256:", "sha256-"), ".imgpkg", "images.yml")
		assert.FileExists(t, outputDirImagesYmlFile)
		actualImagesYmlFile, err = os.ReadFile(outputDirImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- image: %s
kind: ImagesLock
`, randomImageFromPublicRegistry.RefDigest), string(actualImagesYmlFile))
	})
}

func TestPullNestedBundlesLocalizesImagesLockFileWithLocationOCI(t *testing.T) {
	fakeUI := &bundlefakes.FakeUI{}
	pullNestedBundles := true

	t.Run("bundle referencing another bundle in the same repo updates both bundle's imageslock", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()

		randomImageColocatedWithIcecreamBundle := fakeRegistry.WithRandomImage("icecream/bundle")
		icecreamBundle := fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImageColocatedWithIcecreamBundle.RefDigest},
		})
		relocatedIcecreamBundle := fakeRegistry.WithImage("repo/bundle-with-collocated-bundles", icecreamBundle.Image)
		relocatedImageInIcecreamBundle := fakeRegistry.WithImage("repo/bundle-with-collocated-bundles", randomImageColocatedWithIcecreamBundle.Image)

		rootBundle := fakeRegistry.WithBundleFromPath("repo/bundle-with-collocated-bundles", "test_assets/bundle_icecream_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: icecreamBundle.RefDigest, Annotations: map[string]string{"hello": "world"}},
		})

		locationPath, err := os.MkdirTemp(os.TempDir(), "test-location-path")
		assert.NoError(t, err)
		defer os.Remove(locationPath)

		locationForRootBundle := bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{
				{
					Image:    icecreamBundle.RefDigest,
					IsBundle: true,
				},
			},
		}

		locationForNestedBundle := bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{
				{
					Image:    randomImageColocatedWithIcecreamBundle.RefDigest,
					IsBundle: false,
				},
			},
		}

		fakeRegistry.WithLocationsImage("repo/bundle-with-collocated-bundles@"+rootBundle.Digest, locationPath, locationForRootBundle)
		fakeRegistry.WithLocationsImage("repo/bundle-with-collocated-bundles@"+relocatedIcecreamBundle.Digest, locationPath, locationForNestedBundle)

		subject := bundle.NewBundle(rootBundle.RefDigest, fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		rootBundleImagesYmlFile := filepath.Join(outputPath, ".imgpkg", "images.yml")
		assert.FileExists(t, rootBundleImagesYmlFile)
		rootImagesYmlFile, err := os.ReadFile(rootBundleImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- annotations:
    hello: world
  image: %s
kind: ImagesLock
`, relocatedIcecreamBundle.RefDigest), string(rootImagesYmlFile))

		outputDirImagesYmlFile := filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(relocatedIcecreamBundle.Digest, "sha256:", "sha256-"), ".imgpkg", "images.yml")
		assert.FileExists(t, outputDirImagesYmlFile)
		nestedImagesYmlFile, err := os.ReadFile(outputDirImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- image: %s
kind: ImagesLock
`, relocatedImageInIcecreamBundle.RefDigest), string(nestedImagesYmlFile))
	})

	t.Run("bundle referencing another bundle (without a LocationOCI) in the same repo updates both bundle's imageslock", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()

		randomImageColocatedWithIcecreamBundle := fakeRegistry.WithRandomImage("icecream/bundle")
		icecreamBundle := fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImageColocatedWithIcecreamBundle.RefDigest},
		})
		relocatedIcecreamBundle := fakeRegistry.WithImage("repo/bundle-with-collocated-bundles", icecreamBundle.Image)
		relocatedImageInIcecreamBundle := fakeRegistry.WithImage("repo/bundle-with-collocated-bundles", randomImageColocatedWithIcecreamBundle.Image)

		rootBundle := fakeRegistry.WithBundleFromPath("repo/bundle-with-collocated-bundles", "test_assets/bundle_icecream_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: icecreamBundle.RefDigest, Annotations: map[string]string{"hello": "world"}},
		})

		locationPath, err := os.MkdirTemp(os.TempDir(), "test-location-path")
		assert.NoError(t, err)
		defer os.Remove(locationPath)

		locationForRootBundle := bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{
				{
					Image:    icecreamBundle.RefDigest,
					IsBundle: true,
				},
			},
		}

		fakeRegistry.WithLocationsImage("repo/bundle-with-collocated-bundles@"+rootBundle.Digest, locationPath, locationForRootBundle)

		subject := bundle.NewBundle(rootBundle.RefDigest, fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		rootBundleImagesYmlFile := filepath.Join(outputPath, ".imgpkg", "images.yml")
		assert.FileExists(t, rootBundleImagesYmlFile)
		rootImagesYmlFile, err := os.ReadFile(rootBundleImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- annotations:
    hello: world
  image: %s
kind: ImagesLock
`, relocatedIcecreamBundle.RefDigest), string(rootImagesYmlFile))

		outputDirImagesYmlFile := filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(relocatedIcecreamBundle.Digest, "sha256:", "sha256-"), ".imgpkg", "images.yml")
		assert.FileExists(t, outputDirImagesYmlFile)
		nestedImagesYmlFile, err := os.ReadFile(outputDirImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- image: %s
kind: ImagesLock
`, relocatedImageInIcecreamBundle.RefDigest), string(nestedImagesYmlFile))
	})

	t.Run("bundle (without a LocationOCI) referencing another bundle in the same repo updates both bundle's imageslock", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()

		randomImageColocatedWithIcecreamBundle := fakeRegistry.WithRandomImage("icecream/bundle")
		icecreamBundle := fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImageColocatedWithIcecreamBundle.RefDigest},
		})
		relocatedIcecreamBundle := fakeRegistry.WithImage("repo/bundle-with-collocated-bundles", icecreamBundle.Image)
		relocatedImageInIcecreamBundle := fakeRegistry.WithImage("repo/bundle-with-collocated-bundles", randomImageColocatedWithIcecreamBundle.Image)

		rootBundle := fakeRegistry.WithBundleFromPath("repo/bundle-with-collocated-bundles", "test_assets/bundle_icecream_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: icecreamBundle.RefDigest, Annotations: map[string]string{"hello": "world"}},
		})

		locationPath, err := os.MkdirTemp(os.TempDir(), "test-location-path")
		assert.NoError(t, err)
		defer os.Remove(locationPath)

		locationForNestedBundle := bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{
				{
					Image:    randomImageColocatedWithIcecreamBundle.RefDigest,
					IsBundle: false,
				},
			},
		}

		fakeRegistry.WithLocationsImage("repo/bundle-with-collocated-bundles@"+relocatedIcecreamBundle.Digest, locationPath, locationForNestedBundle)

		subject := bundle.NewBundle(rootBundle.RefDigest, fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		rootBundleImagesYmlFile := filepath.Join(outputPath, ".imgpkg", "images.yml")
		assert.FileExists(t, rootBundleImagesYmlFile)
		rootImagesYmlFile, err := os.ReadFile(rootBundleImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- annotations:
    hello: world
  image: %s
kind: ImagesLock
`, relocatedIcecreamBundle.RefDigest), string(rootImagesYmlFile))

		outputDirImagesYmlFile := filepath.Join(outputPath, ".imgpkg", "bundles", strings.ReplaceAll(relocatedIcecreamBundle.Digest, "sha256:", "sha256-"), ".imgpkg", "images.yml")
		assert.FileExists(t, outputDirImagesYmlFile)
		nestedImagesYmlFile, err := os.ReadFile(outputDirImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- image: %s
kind: ImagesLock
`, relocatedImageInIcecreamBundle.RefDigest), string(nestedImagesYmlFile))
	})

	t.Run("bundle referencing only images", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()

		randomImageColocatedWithRootBundle := fakeRegistry.WithRandomImage("repo/root-bundle")
		rootBundle := fakeRegistry.WithBundleFromPath("repo/root-bundle", "test_assets/bundle_icecream_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImageColocatedWithRootBundle.RefDigest},
		})

		locationPath, err := os.MkdirTemp(os.TempDir(), "test-location-path")
		assert.NoError(t, err)
		defer os.Remove(locationPath)

		nestedImageRef := "repo/root-bundle@" + randomImageColocatedWithRootBundle.Digest

		locs := bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{
				{
					Image:    fakeRegistry.ReferenceOnTestServer(nestedImageRef),
					IsBundle: false,
				},
			},
		}

		fakeRegistry.WithLocationsImage("repo/root-bundle@"+rootBundle.Digest, locationPath, locs)

		subject := bundle.NewBundle(rootBundle.RefDigest, fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		rootBundleImagesYmlFile := filepath.Join(outputPath, ".imgpkg", "images.yml")
		assert.FileExists(t, rootBundleImagesYmlFile)
		rootImagesYmlFile, err := os.ReadFile(rootBundleImagesYmlFile)
		assert.NoError(t, err)

		assert.Equal(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- image: %s
kind: ImagesLock
`, fakeRegistry.ReferenceOnTestServer(nestedImageRef)), string(rootImagesYmlFile))

	})
}

func TestPullBundleOutputToUser(t *testing.T) {
	pullNestedBundles := false

	t.Run("bundle referencing an image", func(t *testing.T) {
		output := bytes.NewBufferString("")
		writerUI := ui.NewWriterUI(output, output, nil)

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
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
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
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
	output := bytes.NewBufferString("")
	writerUI := ui.NewWriterUI(output, output, nil)

	t.Run("bundle referencing another collocated bundle", func(t *testing.T) {
		defer output.Reset()

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()

		randomImageColocatedWithIcecreamBundle := fakeRegistry.WithRandomImage("icecream/bundle")
		randomImageFromPrivateRegistry := fakeRegistry.WithImage("library/image1", randomImageColocatedWithIcecreamBundle.Image)

		icecreamBundle := fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImageFromPrivateRegistry.RefDigest},
		})

		fakeRegistry.WithImage("repo/bundle-with-collocated-bundles", icecreamBundle.Image)
		fakeRegistry.WithImage("repo/bundle-with-collocated-bundles", randomImageColocatedWithIcecreamBundle.Image)

		rootBundle := fakeRegistry.WithBundleFromPath("repo/bundle-with-collocated-bundles", "test_assets/bundle_icecream_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: icecreamBundle.RefDigest},
		})

		subject := bundle.NewBundle(rootBundle.RefDigest, fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, writerUI, pullNestedBundles)
		assert.NoError(t, err)

		assert.Regexp(t,
			fmt.Sprintf(`Pulling bundle .*
  Extracting layer .*

Nested bundles
  Pulling nested bundle .*
    Extracting layer .*

Locating image lock file images...
The bundle repo \(%s\) is hosting every image specified in the bundle's Images Lock file \(\.imgpkg/images\.yml\)
`, fakeRegistry.ReferenceOnTestServer("repo/bundle-with-collocated-bundles")), output.String())
	})

	t.Run("bundle referencing another *not* colocated bundle", func(t *testing.T) {
		defer output.Reset()

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
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
		defer output.Reset()

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
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
		defer output.Reset()

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
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

func TestNoteCopy(t *testing.T) {
	t.Run("should succeed if ImageLocations image already exists and immutable error is returned", func(t *testing.T) {
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()

		randomImageColocatedWithIcecreamBundle := fakeRegistry.WithRandomImage("icecream/bundle")

		rootBundle := fakeRegistry.WithBundleFromPath("repo/bundle-with-collocated-bundles", "test_assets/bundle_icecream_with_single_bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImageColocatedWithIcecreamBundle.RefDigest},
		})

		locationPath, err := os.MkdirTemp(os.TempDir(), "test-location-path")
		assert.NoError(t, err)
		defer os.Remove(locationPath)

		locationForRootBundle := bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{
				{
					Image: "some-image-ref-not-matching-root-bundle-resulting-in-diff-sha",
				},
			},
		}

		fakeRegistry.WithLocationsImage("repo/bundle-with-collocated-bundles@"+rootBundle.Digest, locationPath, locationForRootBundle)
		reg := fakeRegistry.Build()

		rootBundleHash, err := regv1.NewHash(rootBundle.Digest)
		assert.NoError(t, err)

		locationsImageTag := fmt.Sprintf("%s-%s.image-locations.imgpkg", rootBundleHash.Algorithm, rootBundleHash.Hex)
		fakeRegistry.WithImmutableTags("repo/bundle-with-collocated-bundles", locationsImageTag)
		defer fakeRegistry.ResetHandler()

		confUI := goui.NewConfUI(goui.NewNoopLogger())
		defer confUI.Flush()
		uiLogger := util.NewUILevelLogger(util.LogDebug, confUI)

		subject := bundle.NewBundleFromPlainImage(plainimage.NewFetchedPlainImageWithTag(rootBundle.RefDigest, "", rootBundle.Image), reg)
		_, _, err = subject.AllImagesRefs(1, uiLogger)
		assert.NoError(t, err)

		processedImages := imageset.NewProcessedImages()
		processedImages.Add(imageset.ProcessedImage{
			UnprocessedImageRef: imageset.UnprocessedImageRef{
				DigestRef: rootBundle.RefDigest,
			},
			DigestRef: rootBundle.RefDigest,
			Image:     rootBundle.Image,
		})
		processedImages.Add(imageset.ProcessedImage{
			UnprocessedImageRef: imageset.UnprocessedImageRef{
				DigestRef: randomImageColocatedWithIcecreamBundle.RefDigest,
			},
			DigestRef: randomImageColocatedWithIcecreamBundle.RefDigest,
			Image:     randomImageColocatedWithIcecreamBundle.Image,
		})

		err = subject.NoteCopy(processedImages, reg, uiLogger)
		assert.NoError(t, err)
	})
}
