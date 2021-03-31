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

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	os.Exit(m.Run())
}

func TestToTarBundle(t *testing.T) {
	bundleName := "library/bundle"
	fakeRegistry := helpers.NewFakeRegistry(t)
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
	})
}

func TestToTarBundleContainingNonDistributableLayers(t *testing.T) {
	bundleName := "library/bundle"
	fakeRegistry := helpers.NewFakeRegistry(t)
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

		require.Regexp(t, "Skipped layer due to it being non-distributable\\. If you would like to include non-distributable layers, use the --include-non-distributable flag", stdOut)
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

	t.Run("When Include-non-distributable flag is provided a warning message should not be printed", func(t *testing.T) {
		stdOut.Reset()
		subject := subject
		subject.IncludeNonDistributable = true

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assert.NotContains(t, stdOut.String(), "Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.")
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
	fakeRegistry := helpers.NewFakeRegistry(t)
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

		assert.Contains(t, stdOut.String(), "Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.\n")
	})
}

func TestToTarImageContainingNonDistributableLayers(t *testing.T) {
	imageName := "library/image"
	fakeRegistry := helpers.NewFakeRegistry(t)
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
	t.Run("When Include-non-distributable flag is provided the tarball should contain every layer", func(t *testing.T) {
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
	imageName := "library/image"
	fakeRegistry := helpers.NewFakeRegistry(t)
	fakeRegistry.WithARandomImageIndex(imageName)
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
		if err != nil {
			t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
		}

		assertTarballContainsEveryLayer(t, imageTarPath)
	})
	t.Run("When Include-non-distributable flag is provided the tarball should contain every layer", func(t *testing.T) {
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
	t.Run("When Include-non-distributable flag is provided a warning message should be printed", func(t *testing.T) {
		stdOut.Reset()
		subject := subject
		subject.IncludeNonDistributable = true

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

func TestToRepoBundleContainingANestedBundle(t *testing.T) {
	bundleName := "library/bundle"
	fakeRegistry := helpers.NewFakeRegistry(t)
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
		processedImageDigest := []string{}
		for _, processedImage := range processedImages.All() {
			processedImageDigest = append(processedImageDigest, processedImage.DigestRef)
		}
		assert.ElementsMatch(t, processedImageDigest, []string{
			destRepo + "@" + bundleWithNestedBundle.Digest,
			destRepo + "@" + bundleWithTwoImages.Digest,
			destRepo + "@" + randomImage.Digest,
			destRepo + "@" + randomImage2.Digest,
		})

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
		processedImageDigest := []string{}
		for _, processedImage := range processedImages.All() {
			processedImageDigest = append(processedImageDigest, processedImage.DigestRef)
		}
		assert.ElementsMatch(t, processedImageDigest, []string{
			destRepo + "@" + bundleWithNestedBundle.Digest,
			destRepo + "@" + bundleWithTwoImages.Digest,
			destRepo + "@" + randomImage.Digest,
			destRepo + "@" + randomImage2.Digest,
		})

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

func TestToRepoBundleWithMultipleRegistries(t *testing.T) {
	fakeDockerhubRegistry := helpers.NewFakeRegistry(t)
	defer fakeDockerhubRegistry.CleanUp()
	fakePrivateRegistry := helpers.NewFakeRegistry(t)
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
	fakeRegistry := helpers.NewFakeRegistry(t)
	image1 := fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config", map[string]string{})
	defer fakeRegistry.CleanUp()
	subject := subject
	subject.ImageFlags = ImageFlags{
		fakeRegistry.ReferenceOnTestServer(imageName),
	}

	t.Run("When Include-non-distributable flag is provided a warning message should be printed", func(t *testing.T) {
		stdOut.Reset()
		subject := subject
		subject.registry = fakeRegistry.Build()
		subject.IncludeNonDistributable = true

		_, err := subject.CopyToRepo(fakeRegistry.ReferenceOnTestServer("fakeregistry/some-repo"))
		if err != nil {
			t.Fatalf("Expected CopyToRepo() to succeed but got: %s", err)
		}

		if !strings.HasSuffix(stdOut.String(), "Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.\n") {
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
