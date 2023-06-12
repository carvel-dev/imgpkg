// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

func TestCopyTarSrc(t *testing.T) {
	logger := helpers.Logger{}

	t.Run("When a bundle contains an ImageIndex", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()
		var expectedNumOfImagesInImageIndex int64 = 3
		imageIndex := fakeRegistry.WithARandomImageIndex("repo/imageindex", expectedNumOfImagesInImageIndex)
		bundleInfo := fakeRegistry.WithBundleFromPath("repo/bundle", "assets/bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: imageIndex.RefDigest},
		})

		fakeRegistry.Build()

		tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
		tempBundleTarFile := filepath.Join(tempBundleTarDir, "bundle-tar.tgz")
		imgpkg.Run([]string{"copy", "-b", bundleInfo.RefDigest, "--to-tar", tempBundleTarFile})
		imgpkg.Run([]string{"copy", "--tar", tempBundleTarFile, "--to-repo", fakeRegistry.ReferenceOnTestServer("copied-bundle")})

		logger.Section("assert ImageIndex were written to dest repo", func() {
			imageIndexRef, err := name.NewDigest(fakeRegistry.ReferenceOnTestServer("copied-bundle") + "@" + imageIndex.Digest)
			assert.NoError(t, err)
			imageIndexGet, err := remote.Get(imageIndexRef)
			assert.NoError(t, err)

			index, err := imageIndexGet.ImageIndex()
			assert.NoError(t, err)
			manifest, err := index.IndexManifest()
			assert.NoError(t, err)
			assert.Len(t, manifest.Manifests, int(expectedNumOfImagesInImageIndex))
			for _, descriptor := range manifest.Manifests {
				digest, err := name.NewDigest(fakeRegistry.ReferenceOnTestServer("copied-bundle") + "@" + descriptor.Digest.String())
				assert.NoError(t, err)
				_, err = remote.Head(digest)
				assert.NoError(t, err)
			}
		})
	})

	t.Run("When a bundle contains an ImageIndex that contains an image with a non-distributable layer", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()
		outputBuffer := bytes.NewBufferString("")

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()

		imageWithNonDistributableLayer, nonDistLayer := fakeRegistry.WithRandomImage("repo/image_belonging_to_image_index").WithNonDistributableLayer()
		nestedImageIndex := fakeRegistry.WithImageIndex("repo/nestedimageindex", imageWithNonDistributableLayer.Image)

		randomImage := fakeRegistry.WithRandomImage("repo/randomimage")
		imageIndex := fakeRegistry.WithImageIndex("repo/imageindex", nestedImageIndex.ImageIndex, randomImage.Image)

		bundleInfo := fakeRegistry.WithBundleFromPath("repo/bundle", "assets/bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: imageIndex.RefDigest},
		})

		fakeRegistry.Build()

		tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
		tempBundleTarFile := filepath.Join(tempBundleTarDir, "bundle-tar.tgz")
		imgpkg.Run([]string{"copy", "-b", bundleInfo.RefDigest, "--to-tar", tempBundleTarFile})
		imgpkg.RunWithOpts([]string{"copy", "--tty", "--tar", tempBundleTarFile, "--to-repo", fakeRegistry.ReferenceOnTestServer("copied-bundle")}, helpers.RunOpts{
			AllowError:   false,
			StderrWriter: outputBuffer,
			StdoutWriter: outputBuffer,
		})

		digest, err := nonDistLayer.Digest()
		require.NoError(t, err)

		ref, err := name.ParseReference(fakeRegistry.ReferenceOnTestServer("copied-bundle"))
		require.NoError(t, err)
		expectedOutput := fmt.Sprintf(`Skipped the followings layer(s) due to it being non-distributable. If you would like to include non-distributable layers, use the --include-non-distributable-layers flag
copy |  - Image: %s
copy |    Layers:
copy |      - %s`, ref.Context().Digest(imageWithNonDistributableLayer.Digest).String(), digest.String())
		assert.Contains(t, outputBuffer.String(), expectedOutput)
	})

	t.Run("When an ImageIndex", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()
		var expectedNumOfImagesInImageIndex int64 = 3
		imageIndex := fakeRegistry.WithARandomImageIndex("repo/imageindex", expectedNumOfImagesInImageIndex)

		fakeRegistry.Build()

		tempTarDir := env.Assets.CreateTempFolder("bundle-tar")
		tempTarFile := filepath.Join(tempTarDir, "bundle-tar.tgz")

		imgpkg.Run([]string{"copy", "-i", imageIndex.RefDigest, "--to-tar", tempTarFile})
		imgpkg.Run([]string{"copy", "--tar", tempTarFile, "--to-repo", fakeRegistry.ReferenceOnTestServer("copied-bundle")})

		logger.Section("assert ImageIndex were written to dest repo", func() {
			imageIndexRef, err := name.NewDigest(fakeRegistry.ReferenceOnTestServer("copied-bundle") + "@" + imageIndex.Digest)
			assert.NoError(t, err)
			imageIndexGet, err := remote.Get(imageIndexRef)
			assert.NoError(t, err)

			index, err := imageIndexGet.ImageIndex()
			assert.NoError(t, err)
			manifest, err := index.IndexManifest()
			assert.NoError(t, err)
			assert.Len(t, manifest.Manifests, int(expectedNumOfImagesInImageIndex))
			for _, descriptor := range manifest.Manifests {
				digest, err := name.NewDigest(fakeRegistry.ReferenceOnTestServer("copied-bundle") + "@" + descriptor.Digest.String())
				assert.NoError(t, err)
				_, err = remote.Head(digest)
				assert.NoError(t, err)
			}
		})
	})

	t.Run("When a bundle contains an image that no longer exists on the registry", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer fakeRegistry.CleanUp()
		randomImage := fakeRegistry.WithRandomImage("repo/randomimage")
		bundleInfo := fakeRegistry.WithBundleFromPath("repo/bundle", "assets/bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: randomImage.RefDigest},
		})

		fakeRegistry.Build()

		tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
		tempBundleTarFile := filepath.Join(tempBundleTarDir, "bundle-tar.tgz")
		imgpkg.Run([]string{"copy", "-b", bundleInfo.RefDigest, "--to-tar", tempBundleTarFile})

		fakeRegistry.RemoveImage("repo/randomimage@" + randomImage.Digest)

		imgpkg.Run([]string{"copy", "--tar", tempBundleTarFile, "--to-repo", fakeRegistry.ReferenceOnTestServer("copied-bundle")})
	})

	t.Run("When there is no longer access to the origin registry the copy is successful", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		destinationFakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer destinationFakeRegistry.CleanUp()

		var tempBundleTarFile string
		originFakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		defer originFakeRegistry.CleanUp()
		logger.Section("create tar from bundle", func() {
			randomImage := originFakeRegistry.WithRandomImage("repo/randomimage")
			bundleInfo := originFakeRegistry.WithBundleFromPath("repo/bundle", "assets/bundle").WithImageRefs([]lockconfig.ImageRef{
				{Image: randomImage.RefDigest},
			})

			originFakeRegistry.Build()

			tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
			tempBundleTarFile = filepath.Join(tempBundleTarDir, "bundle-tar.tgz")
			imgpkg.Run([]string{"copy", "-b", bundleInfo.RefDigest, "--to-tar", tempBundleTarFile})
		})

		// Log all the requests done to the origin registry
		requestLog := originFakeRegistry.WithRequestLogging()

		imgpkg.Run([]string{"copy", "--tar", tempBundleTarFile, "--to-repo", destinationFakeRegistry.ReferenceOnTestServer("copied-bundle")})
		require.Equal(t, 0, requestLog.Len(), "Requests where sent to the origin registry")
	})
}
