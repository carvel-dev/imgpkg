// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
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

func TestCopyingBundleToRepoWithMultipleRegistries(t *testing.T) {
	fakeRegistry := NewFakeRegistry(t)
	defer fakeRegistry.CleanUp()

	sourceBundleName := "localregistry.io/library/bundle"
	destinationBundleName := "localregistry.io/library/copied-bundle"
	// test_assets/bundle contains images that live in index.docker.io
	bundlePath := "test_assets/bundle"
	servedContentPath := "test_assets/image_with_config"
	fakeRegistry.WithBundleFromPath(sourceBundleName, bundlePath).WithEveryImageFrom(servedContentPath)

	imagesLockFile, err := lockconfig.NewImagesLockFromPath(filepath.Join(bundlePath, ".imgpkg", "images.yml"))
	require.NoError(t, err)

	// 'simulate' images that the test bundle reference (index.docker.io) localized to the source bundle [localregistry.io/library/bundle]
	for _, imageRef := range imagesLockFile.Images {
		digest := strings.Split(imageRef.Image, "@")[1]
		fakeRegistry.WithImageFromPath(sourceBundleName+"@"+digest, servedContentPath)
	}

	subject := subject
	subject.BundleFlags = BundleFlags{sourceBundleName}
	subject.registry = fakeRegistry.Build()

	t.Run("Images are copied from localregistry.io and not from the bundle's ImagesLockFile registry (index.docker.io)", func(t *testing.T) {
		processedImages, err := subject.CopyToRepo(destinationBundleName)
		require.NoError(t, err, "expected copy command to succeed")

		require.Len(t, processedImages.All(), 2)
		for _, processedImage := range processedImages.All() {
			assert.Contains(t, processedImage.UnprocessedImageRef.DigestRef, sourceBundleName)
		}
	})
}

func TestCopyingFromImageLockFile(t *testing.T) {
	fakeRegistry := NewFakeRegistry(t)
	defer fakeRegistry.CleanUp()

	destinationImageName := "localregistry.io/library/copied-img"
	imageLockYAML := `apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: some.registry.io/image-1@sha256:9c758bb5cd8a130fc25de0544473ea7e2978ca23dcd78d14d53d30e7b4eef423
  annotations:
    my-annotation: first-image
- image: some.registry.io/image-2@sha256:08075264ab954309ef1382a00157820922599687f99dccd8d8e36d78dc3573d7
  annotations:
    my-annotation: second-image
`
	lockFile, err := ioutil.TempFile("", "images.lock.yml")
	require.NoError(t, err)
	err = ioutil.WriteFile(lockFile.Name(), []byte(imageLockYAML), 0600)
	require.NoError(t, err)

	allImages := []string{
		"some.registry.io/image-1@sha256:9c758bb5cd8a130fc25de0544473ea7e2978ca23dcd78d14d53d30e7b4eef423",
		"some.registry.io/image-2@sha256:08075264ab954309ef1382a00157820922599687f99dccd8d8e36d78dc3573d7",
	}

	expectedImg1 := fakeRegistry.WithImageFromPath(allImages[0], "test_assets/bundle")
	expectedImg2 := fakeRegistry.WithImageFromPath(allImages[1], "test_assets/bundle_with_mult_images")

	subject := subject
	subject.LockInputFlags.LockFilePath = lockFile.Name()
	subject.registry = fakeRegistry.Build()

	t.Run("Copies both images", func(t *testing.T) {
		processedImages, err := subject.CopyToRepo(destinationImageName)
		require.NoError(t, err)

		require.Len(t, processedImages.All(), 2)

		img1 := processedImages.All()[0]
		img2 := processedImages.All()[1]
		assert.Equal(t, expectedImg1.RefDigest, img1.UnprocessedImageRef.DigestRef)
		assert.Equal(t, expectedImg2.RefDigest, img2.UnprocessedImageRef.DigestRef)
	})
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
		require.NoError(t, err)

		assertTarballContainsEveryLayer(t, bundleTarPath)
	})
}

func TestCopyingToTarBundleContainingNonDistributableLayers(t *testing.T) {
	bundleName := "index.docker.io/library/bundle"
	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle_with_mult_images").
		WithEveryImageFrom("test_assets/image_with_config").
		WithNonDistributableLayerInImage("index.docker.io/library/image_with_non_distributable_layer@sha256:555555555555fae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0")
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
		subject.IncludeNonDistributableFlag = IncludeNonDistributableFlag{
			IncludeNonDistributable: true,
		}

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assertTarballContainsEveryLayer(t, imageTarPath)
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
		require.NoError(t, err)

		assert.NotContains(t, stdOut.String(), "Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.")
		assert.NotContains(t, stdOut.String(), "Skipped layer due to it being non-distributable.")
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
		require.NoError(t, err)

		assertTarballContainsEveryLayer(t, imageTarPath)
	})

	t.Run("When Include-non-distributable flag is provided the tarball should contain every layer", func(t *testing.T) {
		subject := subject
		subject.IncludeNonDistributableFlag = IncludeNonDistributableFlag{
			IncludeNonDistributable: true,
		}

		imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
		defer os.Remove(imageTarPath)

		err := subject.CopyToTar(imageTarPath)
		require.NoError(t, err)

		assertTarballContainsEveryLayer(t, imageTarPath)
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
		require.NoError(t, err)

		assert.Contains(t, stdOut.String(), "Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.\n")
	})
}

func TestCopyingToTarImageIndexContainingOnlyDistributableLayers(t *testing.T) {
	imageName := "index.docker.io/library/image"
	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithARandomImageIndex(imageName)
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

		assertTarballContainsEveryLayer(t, imageTarPath)
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

		assertTarballContainsEveryLayer(t, imageTarPath)
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

		assertTarballContainsEveryLayer(t, imageTarPath)
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

	t.Run("When Include-non-distributable flag is provided a warning message should be printed", func(t *testing.T) {
		stdOut.Reset()
		subject := subject
		subject.registry = fakeRegistry.Build()
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

	t.Run("Every layer in the image is mounted", func(t *testing.T) {
		fakeRegistry := NewFakeRegistry(t)
		fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config")
		defer fakeRegistry.CleanUp()

		subject := subject
		fakeReg := fakeRegistry.Build()
		subject.registry = fakeReg

		reference, err := name.ParseReference(imageName)
		if err != nil {
			t.Fatalf("Failed to parse %s as a reference: %v", imageName, err)
		}
		descriptor, err := fakeReg.Get(reference)
		if err != nil {
			t.Fatalf("Failed to fakeReg.Get the given reference: %v", err)
		}
		mountableImage, err := descriptor.Image()
		if err != nil {
			t.Fatalf("Failed to convert the descriptor to a mountableImage: %v", err)
		}

		digest, err := mountableImage.Digest()
		if err != nil {
			t.Fatalf(err.Error())
		}
		referenceNameOfCopiedImage, err := name.ParseReference("index.docker.io/other-repo/image:imgpkg-sha256-" + digest.Hex)
		if err != nil {
			t.Fatalf(err.Error())
		}

		_, err = subject.CopyToRepo("index.docker.io/other-repo/image")
		if err != nil {
			t.Fatalf("Failed to copy to repo: %v", err)
		}
		multiWriteArgsForCall, _ := fakeReg.MultiWriteArgsForCall(0)
		if !reflect.DeepEqual(multiWriteArgsForCall[referenceNameOfCopiedImage], mountableImage) {
			t.Fatalf("Called MultiWrite with key %s unexpected value %v", referenceNameOfCopiedImage, multiWriteArgsForCall)
		}
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

			assert.Truef(t, doesLayerExistInTarball(t, imageTarPath, digest), "did not find the expected layer [%s]", digest)
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
