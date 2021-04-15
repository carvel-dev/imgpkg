// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
)

func TestCopyTarSrc(t *testing.T) {
	t.Run("When a tar contains an ImageIndex", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()

		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()
		imageIndex := fakeRegistry.WithARandomImageIndex("repo/imageindex")
		bundleInfo := fakeRegistry.WithBundleFromPath("repo/bundle", "assets/bundle").WithImageRefs([]lockconfig.ImageRef{
			{Image: imageIndex.RefDigest},
		})

		fakeRegistry.Build()

		tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
		tempBundleTarFile := filepath.Join(tempBundleTarDir, "bundle-tar.tgz")
		imgpkg.Run([]string{"copy", "-b", bundleInfo.RefDigest, "--to-tar", tempBundleTarFile})
		imgpkg.Run([]string{"copy", "--tar", tempBundleTarFile, "--to-repo", fakeRegistry.ReferenceOnTestServer("copied-bundle")})
	})

	t.Run("When a tar contains an ImageIndex that contains an image with a non-distributable layer", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()
		outputBuffer := bytes.NewBufferString("")

		fakeRegistry := helpers.NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		imageWithNonDistributableLayer := fakeRegistry.WithRandomImage("repo/image_belonging_to_image_index").WithNonDistributableLayer()
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
		imgpkg.RunWithOpts([]string{"copy", "--tar", tempBundleTarFile, "--to-repo", fakeRegistry.ReferenceOnTestServer("copied-bundle")}, helpers.RunOpts{
			AllowError:   false,
			StderrWriter: outputBuffer,
			StdoutWriter: outputBuffer,
		})

		assert.Contains(t, outputBuffer.String(), "Skipped layer due to it being non-distributable. If you would like to include non-distributable layers, use the --include-non-distributable flag")
	})

	t.Run("When a tar contains an image that no longer exists on the registry", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()

		fakeRegistry := helpers.NewFakeRegistry(t)
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
}
