// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	goui "github.com/cppforlife/go-cli-ui/ui"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var subject CopyRepoSrc
var stdOut *bytes.Buffer

func TestMain(m *testing.M) {
	stdOut = bytes.NewBufferString("")
	logger := util.NewLogger(stdOut)
	prefixedLogger := logger.NewPrefixedWriter("test | ")
	levelLogger := logger.NewLevelLogger(util.LogWarn, prefixedLogger)
	imageSet := imageset.NewImageSet(1, prefixedLogger)

	subject = CopyRepoSrc{
		logger:             levelLogger,
		imageSet:           imageSet,
		tarImageSet:        imageset.NewTarImageSet(imageSet, 1, prefixedLogger),
		Concurrency:        1,
		signatureRetriever: &fakeSignatureRetriever{},
	}

	os.Exit(m.Run())
}

func TestToTarBundle(t *testing.T) {
	bundleName := "library/bundle"
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	bundleWithImages := fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle").
		WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})

	defer fakeRegistry.CleanUp()

	subject := subject
	subject.BundleFlags = BundleFlags{fakeRegistry.ReferenceOnTestServer(bundleName)}
	subject.registry = fakeRegistry.Build()

	t.Run("Tar should contain every layer", func(t *testing.T) {
		bundleTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(bundleTarPath)

		err := subject.CopyToTar(bundleTarPath)
		require.NoError(t, err)

		assertTarballContainsEveryLayer(t, bundleTarPath)
	})

	t.Run("When a bundle contains a bundle, it copies all layers to tar", func(t *testing.T) {
		assets := &helpers.Assets{T: t}
		defer assets.CleanCreatedFolders()

		bundleWithNested := fakeRegistry.
			WithBundleFromPath("library/with-nested-bundle", "test_assets/bundle").
			WithImageRefs([]lockconfig.ImageRef{
				{Image: bundleWithImages.RefDigest},
			})

		bundleLock, err := lockconfig.NewBundleLockFromBytes([]byte(fmt.Sprintf(`
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: BundleLock
bundle:
  image: %s
`, bundleWithNested.RefDigest)))
		assert.NoError(t, err)

		bundleLockTempDir := filepath.Join(assets.CreateTempFolder("bundle-lock"), "lock.yml")
		assert.NoError(t, bundleLock.WriteToPath(bundleLockTempDir))

		subject := subject
		subject.BundleFlags.Bundle = ""
		subject.LockInputFlags.LockFilePath = bundleLockTempDir
		subject.registry = fakeRegistry.Build()

		subject.BundleFlags.Bundle = fakeRegistry.ReferenceOnTestServer(
			bundleWithNested.BundleName + "@" + bundleWithNested.Digest)

		tarDir := assets.CreateTempFolder("tar-copy")
		imageTarPath := filepath.Join(tarDir, "bundle.tar")

		err = subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assertTarballContainsOnlyDistributableLayers(imageTarPath, t)

		assertTarballLabelsOuterBundle(imageTarPath, subject.BundleFlags.Bundle, t)
	})
}

func TestToTarBundleContainingNonDistributableLayers(t *testing.T) {
	bundleName := "library/bundle"
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	defer fakeRegistry.CleanUp()
	randomImage := fakeRegistry.WithRandomImage("library/image_with_config")
	randomImageWithNonDistributableLayer := fakeRegistry.
		WithRandomImage("library/image_with_non_dist_layer").WithNonDistributableLayer()

	fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle_with_mult_images").
		WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImage.RefDigest},
			{Image: randomImageWithNonDistributableLayer.RefDigest},
		})

	subject := subject
	subject.BundleFlags = BundleFlags{fakeRegistry.ReferenceOnTestServer(bundleName)}
	subject.registry = fakeRegistry.Build()

	t.Run("Tar should contain every distributable layer only", func(t *testing.T) {
		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assertTarballContainsOnlyDistributableLayers(imageTarPath, t)
	})

	t.Run("Warning message should be printed indicating layers have been skipped", func(t *testing.T) {
		stdOut.Reset()

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		require.Regexp(t, "Skipped layer due to it being non-distributable\\. If you would like to include non-distributable layers, use the --include-non-distributable-layers flag", stdOut)
	})

	t.Run("When Include-non-distributable-layers flag is provided the tarball should contain every layer", func(t *testing.T) {
		subject := subject
		subject.IncludeNonDistributable = true

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assertTarballContainsEveryLayer(t, imageTarPath)
	})

	t.Run("When Include-non-distributable-layers flag is provided a warning message should not be printed", func(t *testing.T) {
		stdOut.Reset()
		subject := subject
		subject.IncludeNonDistributable = true

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assert.NotContains(t, stdOut.String(), "Warning: '--include-non-distributable-layers' flag provided, but no images contained a non-distributable layer.")
		assert.NotContains(t, stdOut.String(), "Skipped layer due to it being non-distributable.")
	})

	t.Run("When a bundle contains a bundle with non distributable layer, it copies all layers to tar", func(t *testing.T) {
		assets := &helpers.Assets{T: t}
		bundleBuilder := helpers.NewBundleDir(t, assets)
		defer assets.CleanCreatedFolders()
		imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s@%s
`, fakeRegistry.ReferenceOnTestServer(bundleName), fakeRegistry.
			WithBundleFromPath(bundleName, "test_assets/bundle_with_mult_images").Digest)
		bundleDir := bundleBuilder.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
		bundleWithNested := fakeRegistry.WithBundleFromPath("library/with-nested-bundle", bundleDir)

		subject := subject
		subject.registry = fakeRegistry.Build()

		subject.BundleFlags.Bundle = fakeRegistry.ReferenceOnTestServer(bundleWithNested.BundleName + "@" +
			bundleWithNested.Digest)

		tarDir := assets.CreateTempFolder("tar-copy")
		imageTarPath := filepath.Join(tarDir, "bundle.tar")

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assertTarballContainsOnlyDistributableLayers(imageTarPath, t)
	})
}

func TestToTarImage(t *testing.T) {
	imageName := "library/image"
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config", map[string]string{})
	defer fakeRegistry.CleanUp()

	subject := subject
	subject.ImageFlags = ImageFlags{
		fakeRegistry.ReferenceOnTestServer(imageName),
	}
	subject.registry = fakeRegistry.Build()

	t.Run("Tar should contain every layer", func(t *testing.T) {
		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assertTarballContainsEveryLayer(t, imageTarPath)
	})

	t.Run("When Include-non-distributable flag is provided the tarball should contain every layer", func(t *testing.T) {
		subject := subject
		subject.IncludeNonDistributable = true

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assertTarballContainsEveryLayer(t, imageTarPath)
	})

	t.Run("When Include-non-distributable flag is provided a warning message should be printed", func(t *testing.T) {
		stdOut.Reset()
		subject := subject
		subject.IncludeNonDistributable = true

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assert.Contains(t, stdOut.String(), "Warning: '--include-non-distributable-layers' flag provided, but no images contained a non-distributable layer.\n")
	})
}

func TestToTarImageContainingNonDistributableLayers(t *testing.T) {
	imageName := "library/image"
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config", map[string]string{}).
		WithNonDistributableLayer()
	defer fakeRegistry.CleanUp()
	subject := subject
	subject.ImageFlags = ImageFlags{
		fakeRegistry.ReferenceOnTestServer(imageName),
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
	t.Run("When Include-non-distributable-layers flag is provided the tarball should contain every layer", func(t *testing.T) {
		subject := subject
		subject.IncludeNonDistributable = true

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		assertTarballContainsEveryLayer(t, imageTarPath)
	})
}

func TestToTarImageIndex(t *testing.T) {
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	imageIndex := fakeRegistry.WithARandomImageIndex("library/imageindex", 3)
	defer fakeRegistry.CleanUp()

	subject := subject
	subject.ImageFlags = ImageFlags{
		imageIndex.RefDigest,
	}
	subject.registry = fakeRegistry.Build()

	t.Run("should fail with an error message", func(t *testing.T) {
		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		if assert.Error(t, err) {
			assert.Equal(t, "Unable to copy non-images (such as ImageIndexes)", err.Error())
		}
	})
}

func TestToRepoImageIndex(t *testing.T) {
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	randomImageIndex := fakeRegistry.WithARandomImageIndex("library/imageindex", 3)
	defer fakeRegistry.CleanUp()
	subject := subject
	subject.ImageFlags = ImageFlags{
		randomImageIndex.RefDigest,
	}
	destinationImageName := "library/copied-img"

	t.Run("should fail with an error message", func(t *testing.T) {
		subject := subject
		subject.registry = fakeRegistry.Build()

		_, err := subject.CopyToRepo(fakeRegistry.ReferenceOnTestServer(destinationImageName))
		if assert.Error(t, err) {
			assert.Equal(t, "Unable to copy non-images (such as ImageIndexes)", err.Error())
		}
	})

	t.Run("with an ImagesLock file should fail with an error message", func(t *testing.T) {
		assets := &helpers.Assets{T: t}
		defer assets.CleanCreatedFolders()

		imageIndexRefDigest := fakeRegistry.WithARandomImageIndex("library/image-2", 3).RefDigest
		imageLockYAML := fmt.Sprintf(`apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
  annotations:
    my-annotation: first-image
- image: %s
  annotations:
    my-annotation: second-image
`, randomImageIndex.RefDigest, imageIndexRefDigest)
		lockFile, err := ioutil.TempFile(assets.CreateTempFolder("images-lock-dir"), "images.lock.yml")

		require.NoError(t, err)
		err = ioutil.WriteFile(lockFile.Name(), []byte(imageLockYAML), 0600)
		require.NoError(t, err)

		subject := subject
		subject.LockInputFlags.LockFilePath = lockFile.Name()
		subject.registry = fakeRegistry.Build()

		_, err = subject.CopyToRepo(fakeRegistry.ReferenceOnTestServer(destinationImageName))
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Unable to copy non-images (such as ImageIndexes) using an Images Lock file")
		}
	})
}

func TestToRepoBundleContainingANestedBundle(t *testing.T) {
	bundleName := "library/bundle"
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	defer fakeRegistry.CleanUp()
	randomImage := fakeRegistry.WithRandomImage("library/image_with_config")
	randomImage2 := fakeRegistry.WithRandomImage("library/image_with_config_2")

	bundleWithTwoImages := fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle_with_mult_images").
		WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImage.RefDigest},
			{Image: randomImage2.RefDigest},
		})

	bundleWithNestedBundle := fakeRegistry.WithBundleFromPath("library/bundle-with-nested-bundle",
		"test_assets/bundle_with_mult_images").WithImageRefs([]lockconfig.ImageRef{
		{Image: bundleWithTwoImages.RefDigest},
	})

	subject := subject
	subject.BundleFlags.Bundle = bundleWithNestedBundle.RefDigest
	subject.registry = fakeRegistry.Build()

	t.Run("When recursive bundle is enabled, it copies every image to repo", func(t *testing.T) {
		subject := subject
		subject.registry = fakeRegistry.Build()

		destRepo := fakeRegistry.ReferenceOnTestServer("library/bundle-copy")
		processedImages, err := subject.CopyToRepo(destRepo)
		require.NoError(t, err)

		require.Len(t, processedImages.All(), 4)

		var processedBundle imageset.ProcessedImage
		processedImageDigest := []string{}
		for _, processedImage := range processedImages.All() {
			processedImageDigest = append(processedImageDigest, processedImage.DigestRef)
			if _, ok := processedImage.Labels[rootBundleLabelKey]; ok {
				processedBundle = processedImage
			}
		}
		assert.ElementsMatch(t, processedImageDigest, []string{
			destRepo + "@" + bundleWithNestedBundle.Digest,
			destRepo + "@" + bundleWithTwoImages.Digest,
			destRepo + "@" + randomImage.Digest,
			destRepo + "@" + randomImage2.Digest,
		})

		assert.Equal(t, processedBundle.DigestRef, destRepo+"@"+bundleWithNestedBundle.Digest)
	})

	t.Run("When recursive bundle is enabled and a lock file is provided, it copies every image to repo", func(t *testing.T) {
		assets := &helpers.Assets{T: t}
		defer assets.CleanCreatedFolders()
		bundleLock, err := lockconfig.NewBundleLockFromBytes([]byte(fmt.Sprintf(`
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: BundleLock
bundle:
  image: %s
`, bundleWithNestedBundle.RefDigest)))
		assert.NoError(t, err)

		bundleLockTempDir := filepath.Join(assets.CreateTempFolder("bundle-lock"), "lock.yml")
		assert.NoError(t, bundleLock.WriteToPath(bundleLockTempDir))

		subject := subject
		subject.BundleFlags.Bundle = ""
		subject.LockInputFlags.LockFilePath = bundleLockTempDir
		subject.registry = fakeRegistry.Build()

		destRepo := fakeRegistry.ReferenceOnTestServer("library/bundle-copy")
		processedImages, err := subject.CopyToRepo(destRepo)
		require.NoError(t, err)

		require.Len(t, processedImages.All(), 4)

		var processedBundle imageset.ProcessedImage
		processedImageDigest := []string{}
		for _, processedImage := range processedImages.All() {
			processedImageDigest = append(processedImageDigest, processedImage.DigestRef)
			if _, ok := processedImage.Labels[rootBundleLabelKey]; ok {
				processedBundle = processedImage
			}
		}
		assert.ElementsMatch(t, processedImageDigest, []string{
			destRepo + "@" + bundleWithNestedBundle.Digest,
			destRepo + "@" + bundleWithTwoImages.Digest,
			destRepo + "@" + randomImage.Digest,
			destRepo + "@" + randomImage2.Digest,
		})
		assert.Equal(t, processedBundle.DigestRef, destRepo+"@"+bundleWithNestedBundle.Digest)
	})

	t.Run("When recursive bundle is enabled and an images lock file is provided, it returns an error message to the user", func(t *testing.T) {
		assets := &helpers.Assets{T: t}
		defer assets.CleanCreatedFolders()
		imagesLock, err := lockconfig.NewImagesLockFromBytes([]byte(fmt.Sprintf(`
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, bundleWithNestedBundle.RefDigest)))
		require.NoError(t, err)
		imagesLockTempDir := filepath.Join(assets.CreateTempFolder("images-lock"), "images-lock.yml")
		assert.NoError(t, imagesLock.WriteToPath(imagesLockTempDir))

		subject := subject
		subject.BundleFlags.Bundle = ""
		subject.LockInputFlags.LockFilePath = imagesLockTempDir
		subject.registry = fakeRegistry.Build()

		destRepo := fakeRegistry.ReferenceOnTestServer("library/bundle-copy")
		_, err = subject.CopyToRepo(destRepo)
		require.Error(t, err)
		assert.EqualError(t, err, "Unable to copy bundles using an Images Lock file (hint: Create a bundle with these images)")
	})
}

func TestToRepoBundleCreatesValidLocationOCI(t *testing.T) {
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	defer fakeRegistry.CleanUp()

	bundleWithOneImages := fakeRegistry.WithBundleFromPath("library/bundle", "test_assets/bundle_with_mult_images").
		WithImageRefs([]lockconfig.ImageRef{
			{Image: "hello-world@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6"},
		})

	bundleWithNestedBundle := fakeRegistry.WithBundleFromPath("library/bundle-with-nested-bundle",
		"test_assets/bundle_with_mult_images").WithImageRefs([]lockconfig.ImageRef{
		{Image: bundleWithOneImages.RefDigest},
	})

	subject := subject
	subject.BundleFlags.Bundle = bundleWithNestedBundle.RefDigest
	subject.registry = fakeRegistry.Build()

	t.Run("A bundle with an image without a qualified image name", func(t *testing.T) {
		assets := &helpers.Assets{T: t}
		defer assets.CleanCreatedFolders()

		subject := subject
		subject.registry = fakeRegistry.Build()

		destRepo := fakeRegistry.ReferenceOnTestServer("library/bundle-copy")
		processedImages, err := subject.CopyToRepo(destRepo)
		require.NoError(t, err)

		require.Len(t, processedImages.All(), 3)
		var processedBundle imageset.ProcessedImage
		processedImageDigest := []string{}
		for _, processedImage := range processedImages.All() {
			processedImageDigest = append(processedImageDigest, processedImage.DigestRef)
			if _, ok := processedImage.Labels[rootBundleLabelKey]; ok {
				processedBundle = processedImage
			}
		}
		assert.ElementsMatch(t, processedImageDigest, []string{
			destRepo + "@" + bundleWithNestedBundle.Digest,
			destRepo + "@" + bundleWithOneImages.Digest,
			destRepo + "@" + "sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6",
		})
		assert.Equal(t, processedBundle.DigestRef, destRepo+"@"+bundleWithNestedBundle.Digest)

		locationImg := fmt.Sprintf("%s:%s.image-locations.imgpkg", destRepo, strings.ReplaceAll(bundleWithNestedBundle.Digest, ":", "-"))
		refs := []string{locationImg}
		require.NoError(t, validateImagesPresenceInRegistry(t, refs))

		locationImgFolder := assets.CreateTempFolder("locations")
		downloadImagesLocation(t, locationImg, locationImgFolder)

		locationsFilePath := filepath.Join(locationImgFolder, "image-locations.yml")
		require.FileExists(t, locationsFilePath)

		cfg, err := bundle.NewLocationConfigFromPath(locationsFilePath)
		require.NoError(t, err)

		require.Equal(t, bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{{
				Image: bundleWithOneImages.RefDigest,
				// Repository not used for now because all images will be present in the same repository
				IsBundle: true,
			}},
		}, cfg)

		locationImg = fmt.Sprintf("%s:%s.image-locations.imgpkg", destRepo, strings.ReplaceAll(bundleWithOneImages.Digest, ":", "-"))
		refs = []string{locationImg}
		require.NoError(t, validateImagesPresenceInRegistry(t, refs))

		locationImgFolder = assets.CreateTempFolder("locations")
		downloadImagesLocation(t, locationImg, locationImgFolder)

		locationsFilePath = filepath.Join(locationImgFolder, "image-locations.yml")
		require.FileExists(t, locationsFilePath)

		cfg, err = bundle.NewLocationConfigFromPath(locationsFilePath)
		require.NoError(t, err)

		require.Equal(t, bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{{
				Image:    "index.docker.io/library/hello-world@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6",
				IsBundle: false,
			}},
		}, cfg)
	})

	knownStatusCodesMeaningImageWasNotFound := []int{404, 401, 403}
	for _, remappedStatusCode := range knownStatusCodesMeaningImageWasNotFound {
		t.Run(fmt.Sprintf("A registry that returns %d status codes when an image doesn't exist", remappedStatusCode), func(t *testing.T) {
			assets := &helpers.Assets{T: t}
			defer assets.CleanCreatedFolders()

			subject := subject
			subject.registry = fakeRegistry.Build()

			destRepo := fakeRegistry.ReferenceOnTestServer("library/bundle-copy")

			locationImg := fmt.Sprintf("%s:%s.image-locations.imgpkg", destRepo, strings.ReplaceAll(bundleWithNestedBundle.Digest, ":", "-"))
			refs := []string{locationImg}

			fakeRegistry.WithImageStatusCodeRemap(fmt.Sprintf("%s.image-locations.imgpkg", strings.ReplaceAll(bundleWithNestedBundle.Digest, ":", "-")), 404, remappedStatusCode)
			defer fakeRegistry.ResetHandler()

			processedImages, err := subject.CopyToRepo(destRepo)
			require.NoError(t, err)

			require.Len(t, processedImages.All(), 3)

			var processedBundle imageset.ProcessedImage
			processedImageDigest := []string{}
			for _, processedImage := range processedImages.All() {
				processedImageDigest = append(processedImageDigest, processedImage.DigestRef)

				if _, ok := processedImage.Labels[rootBundleLabelKey]; ok {
					processedBundle = processedImage
				}
			}
			assert.ElementsMatch(t, processedImageDigest, []string{
				destRepo + "@" + bundleWithNestedBundle.Digest,
				destRepo + "@" + bundleWithOneImages.Digest,
				destRepo + "@" + "sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6",
			})
			assert.Equal(t, processedBundle.DigestRef, destRepo+"@"+bundleWithNestedBundle.Digest)

			require.NoError(t, validateImagesPresenceInRegistry(t, refs))
		})
	}

}

func TestToRepoBundleRunTwiceCreatesValidLocationOCI(t *testing.T) {
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	defer fakeRegistry.CleanUp()

	bundleWithOneImages := fakeRegistry.WithBundleFromPath("library/bundle", "test_assets/bundle_with_mult_images").
		WithImageRefs([]lockconfig.ImageRef{
			{Image: "hello-world@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6"},
		})

	reference, err := name.ParseReference("hello-world@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6")
	require.NoError(t, err)
	helloworld, err := remote.Get(reference)
	require.NoError(t, err)
	image, err := helloworld.Image()
	require.NoError(t, err)
	fakeRegistry.WithImage("library/bundle", image)

	bundleWithNestedBundle := fakeRegistry.WithBundleFromPath("library/bundle-with-nested-bundle",
		"test_assets/bundle_with_mult_images").WithImageRefs([]lockconfig.ImageRef{
		{Image: bundleWithOneImages.RefDigest},
	})

	subject := subject
	subject.BundleFlags.Bundle = bundleWithNestedBundle.RefDigest
	subject.registry = fakeRegistry.Build()

	t.Run("A bundle with an image without a qualified image name", func(t *testing.T) {
		assets := &helpers.Assets{T: t}
		defer assets.CleanCreatedFolders()

		subject := subject
		subject.registry = fakeRegistry.Build()

		destRepo := fakeRegistry.ReferenceOnTestServer("library/bundle-copy")
		processedImages, err := subject.CopyToRepo(destRepo)
		require.NoError(t, err)

		require.Len(t, processedImages.All(), 3)

		var processedBundle imageset.ProcessedImage
		processedImageDigest := []string{}
		for _, processedImage := range processedImages.All() {
			processedImageDigest = append(processedImageDigest, processedImage.DigestRef)
			if _, ok := processedImage.Labels[rootBundleLabelKey]; ok {
				processedBundle = processedImage
			}
		}
		assert.ElementsMatch(t, processedImageDigest, []string{
			destRepo + "@" + bundleWithNestedBundle.Digest,
			destRepo + "@" + bundleWithOneImages.Digest,
			destRepo + "@" + "sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6",
		})
		assert.Equal(t, processedBundle.DigestRef, destRepo+"@"+bundleWithNestedBundle.Digest)

		locationImg := fmt.Sprintf("%s:%s.image-locations.imgpkg", destRepo, strings.ReplaceAll(bundleWithNestedBundle.Digest, ":", "-"))
		refs := []string{locationImg}
		require.NoError(t, validateImagesPresenceInRegistry(t, refs))

		locationImgFolder := assets.CreateTempFolder("locations")
		downloadImagesLocation(t, locationImg, locationImgFolder)

		locationsFilePath := filepath.Join(locationImgFolder, "image-locations.yml")
		require.FileExists(t, locationsFilePath)

		cfg, err := bundle.NewLocationConfigFromPath(locationsFilePath)
		require.NoError(t, err)

		require.Equal(t, bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{{
				Image: bundleWithOneImages.RefDigest,
				// Repository not used for now because all images will be present in the same repository
				IsBundle: true,
			}},
		}, cfg)

		locationImg = fmt.Sprintf("%s:%s.image-locations.imgpkg", destRepo, strings.ReplaceAll(bundleWithOneImages.Digest, ":", "-"))
		refs = []string{locationImg}
		require.NoError(t, validateImagesPresenceInRegistry(t, refs))

		locationImgFolder = assets.CreateTempFolder("locations")
		downloadImagesLocation(t, locationImg, locationImgFolder)

		locationsFilePath = filepath.Join(locationImgFolder, "image-locations.yml")
		require.FileExists(t, locationsFilePath)

		cfg, err = bundle.NewLocationConfigFromPath(locationsFilePath)
		require.NoError(t, err)

		require.Equal(t, bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{{
				Image:    "index.docker.io/library/hello-world@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6",
				IsBundle: false,
			}},
		}, cfg)
	})
}

func TestToRepoBundleWithMultipleRegistries(t *testing.T) {
	fakeDockerhubRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	defer fakeDockerhubRegistry.CleanUp()
	fakePrivateRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	defer fakePrivateRegistry.CleanUp()

	sourceBundleName := "library/bundle"
	destinationBundleName := "library/copied-bundle"

	randomImage1FromDockerhub := fakeDockerhubRegistry.WithRandomImage("random-image1")
	fakePrivateRegistry.WithImage(sourceBundleName, randomImage1FromDockerhub.Image)

	// test_assets/bundle contains images that live in dockerhub
	bundleWithImageRefsToDockerhub := fakePrivateRegistry.WithBundleFromPath(sourceBundleName,
		"test_assets/bundle_with_dockerhub_images").WithImageRefs([]lockconfig.ImageRef{
		{Image: randomImage1FromDockerhub.RefDigest},
	})

	subject := subject
	subject.BundleFlags = BundleFlags{bundleWithImageRefsToDockerhub.RefDigest}
	subject.registry = fakePrivateRegistry.Build()
	fakeDockerhubRegistry.Build()

	t.Run("Images are copied from fake-registry and not from the bundle's ImagesLockFile registry (index.docker.io)", func(t *testing.T) {
		processedImages, err := subject.CopyToRepo(fakePrivateRegistry.ReferenceOnTestServer(destinationBundleName))
		require.NoError(t, err, "expected copy command to succeed")

		require.Len(t, processedImages.All(), 2)
		for _, processedImage := range processedImages.All() {
			assert.Contains(t, processedImage.UnprocessedImageRef.DigestRef, fakePrivateRegistry.
				ReferenceOnTestServer(sourceBundleName))
		}
	})

	t.Run("Using a BundleLock file, Images are copied from fake-registry and not from the bundle's ImagesLockFile registry (index.docker.io)", func(t *testing.T) {
		assets := &helpers.Assets{T: t}
		defer assets.CleanCreatedFolders()

		bundleLock, err := lockconfig.NewBundleLockFromBytes([]byte(fmt.Sprintf(`
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: BundleLock
bundle:
  image: %s
`, bundleWithImageRefsToDockerhub.RefDigest)))
		assert.NoError(t, err)

		bundleLockTempDir := filepath.Join(assets.CreateTempFolder("bundle-lock"), "lock.yml")
		assert.NoError(t, bundleLock.WriteToPath(bundleLockTempDir))

		subject := subject
		subject.BundleFlags.Bundle = ""
		subject.LockInputFlags.LockFilePath = bundleLockTempDir

		processedImages, err := subject.CopyToRepo(fakePrivateRegistry.ReferenceOnTestServer(destinationBundleName))
		require.NoError(t, err, "expected copy command to succeed")

		require.Len(t, processedImages.All(), 2)
		for _, processedImage := range processedImages.All() {
			assert.Contains(t, processedImage.UnprocessedImageRef.DigestRef, fakePrivateRegistry.
				ReferenceOnTestServer(sourceBundleName))
		}
	})
}

func TestToRepoImage(t *testing.T) {
	imageName := "library/image"
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	image1 := fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config", map[string]string{})
	defer fakeRegistry.CleanUp()
	subject := subject
	subject.ImageFlags = ImageFlags{
		fakeRegistry.ReferenceOnTestServer(imageName),
	}

	t.Run("When Include-non-distributable-layers flag is provided a warning message should be printed", func(t *testing.T) {
		stdOut.Reset()
		subject := subject
		subject.registry = fakeRegistry.Build()
		subject.IncludeNonDistributable = true

		_, err := subject.CopyToRepo(fakeRegistry.ReferenceOnTestServer("fakeregistry/some-repo"))
		if err != nil {
			t.Fatalf("Expected CopyToRepo() to succeed but got: %s", err)
		}

		if !strings.HasSuffix(stdOut.String(), "Warning: '--include-non-distributable-layers' flag provided, but no images contained a non-distributable layer.\n") {
			t.Fatalf("Expected command to give warning message, but got: %s", stdOut.String())
		}
	})

	t.Run("When an ImageLock file is provided it should copy every image from the file", func(t *testing.T) {
		assets := &helpers.Assets{T: t}
		defer assets.CleanCreatedFolders()

		destinationImageName := "library/copied-img"

		image2RefDigest := fakeRegistry.WithRandomImage("library/image-2").RefDigest
		imageLockYAML := fmt.Sprintf(`apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
  annotations:
    my-annotation: first-image
- image: %s
  annotations:
    my-annotation: second-image
`, image1.RefDigest, image2RefDigest)
		lockFile, err := ioutil.TempFile(assets.CreateTempFolder("images-lock-dir"), "images.lock.yml")

		require.NoError(t, err)
		err = ioutil.WriteFile(lockFile.Name(), []byte(imageLockYAML), 0600)
		require.NoError(t, err)

		subject := subject
		subject.LockInputFlags.LockFilePath = lockFile.Name()
		subject.registry = fakeRegistry.Build()

		processedImages, err := subject.CopyToRepo(fakeRegistry.ReferenceOnTestServer(destinationImageName))
		if err != nil {
			t.Fatalf("Expected CopyToRepo() to succeed but got: %s", err)
		}

		require.Len(t, processedImages.All(), 2)

		assert.Equal(t, image1.RefDigest, processedImages.All()[1].UnprocessedImageRef.DigestRef)
		assert.Equal(t, image2RefDigest, processedImages.All()[0].UnprocessedImageRef.DigestRef)

	})

}

type fakeSignatureRetriever struct {
}

func (f fakeSignatureRetriever) Fetch(images *imageset.UnprocessedImageRefs) (*imageset.UnprocessedImageRefs, error) {
	return imageset.NewUnprocessedImageRefs(), nil
}

var _ SignatureRetriever = new(fakeSignatureRetriever)

func assertTarballContainsEveryLayer(t *testing.T, imageTarPath string) {
	path := imagetar.NewTarReader(imageTarPath)
	imageOrIndex, err := path.Read()
	require.NoError(t, err)

	for _, imageInManifest := range imageOrIndex {
		layers, err := (*imageInManifest.Image).Layers()
		require.NoError(t, err)

		for _, layer := range layers {
			digest, err := layer.Digest()
			require.NoError(t, err)

			assert.Truef(t, doesLayerExistInTarball(t, imageTarPath, digest), "did not find the expected layer [%s]",
				digest)
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

			if doesLayerExistInTarball(t, imageTarPath, digest) && !mediaType.IsDistributable() {
				t.Fatalf("Expected to fail. The foreign layer was found in the tarball when we expected it not to")
			}
		}
	}

}

func assertTarballLabelsOuterBundle(imageTarPath string, outerBundleRef string, t *testing.T) {
	tarReader := imagetar.NewTarReader(imageTarPath)
	imageOrIndices, err := tarReader.Read()
	assert.NoError(t, err)
	var imageReferencesFound []imagedesc.ImageOrIndex
	for _, imageOrIndex := range imageOrIndices {
		if _, ok := imageOrIndex.Labels["dev.carvel.imgpkg.copy.root-bundle"]; ok {
			imageReferencesFound = append(imageReferencesFound, imageOrIndex)
		}
	}

	assert.NotNil(t, imageReferencesFound)
	assert.Len(t, imageReferencesFound, 1)
	assert.Equal(t, outerBundleRef, imageReferencesFound[0].Ref())
}

func doesLayerExistInTarball(t *testing.T, path string, digest regv1.Hash) bool {
	filePathInTar := digest.Algorithm + "-" + digest.Hex + ".tar.gz"
	file, err := os.Open(path)
	require.NoError(t, err)
	tf := tar.NewReader(file)
	for {
		hdr, err := tf.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if hdr.Name == filePathInTar {
			return true
		}
	}
	return false
}

func validateImagesPresenceInRegistry(t *testing.T, refs []string) error {
	for _, refString := range refs {
		ref, err := name.ParseReference(refString)
		require.NoError(t, err)
		if _, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
			return fmt.Errorf("validating image %s: %v", refString, err)
		}
	}
	return nil
}

func downloadImagesLocation(t *testing.T, imgRef, location string) {
	imageReg, err := name.ParseReference(imgRef, name.WeakValidation)
	require.NoError(t, err)
	img, err := remote.Image(imageReg, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(t, err)

	output := bytes.NewBufferString("")
	writerUI := goui.NewWriterUI(output, output, nil)
	err = ctlimg.NewDirImage(filepath.Join(location), img, writerUI).AsDirectory()
	require.NoError(t, err)
}
