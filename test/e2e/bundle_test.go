// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"carvel.dev/imgpkg/test/helpers"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushPullBundle(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)

	out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
	digest := helpers.ExtractDigest(t, out)
	outDir := env.Assets.CreateTempFolder("imgpkg-test-basic")

	splits := strings.Split(digest, ":")
	bundleRefWithTag := env.Image + ":" + fmt.Sprintf("%s-%s.imgpkg", splits[0], splits[1])
	imgpkg.Run([]string{"pull", "-b", bundleRefWithTag, "-o", outDir})

	env.Assets.ValidateFilesAreEqual(bundleDir, outDir, []string{
		".imgpkg/bundle.yml",
		".imgpkg/images.yml",
		"README.md",
		"LICENSE",
		"config/config.yml",
		"config/inner-dir/README.txt",
	})
}

func TestBundlePushPullAnnotation(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir})

	ref, _ := name.NewTag(env.Image, name.WeakValidation)
	image, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(t, err)

	config, err := image.ConfigFile()
	require.NoError(t, err)

	require.Contains(t, config.Config.Labels, "dev.carvel.imgpkg.bundle")

	outDir := env.Assets.CreateTempFolder("bundle-annotation")

	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", outDir})

	env.Assets.ValidateFilesAreEqual(bundleDir, outDir, env.Assets.FilesInFolder())
}

func TestPushWithFileExclusion(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)

	env.BundleFactory.AddFileToBundle("excluded-file.txt", "I will not be present in the bundle")
	env.BundleFactory.AddFileToBundle(
		filepath.Join("nested-dir", "excluded-file.txt"),
		"this file will not be excluded because it is nested",
	)

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir, "--file-exclusion", "excluded-file.txt"})

	outDir := env.Assets.CreateTempFolder("bundle-exclusion")
	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", outDir})

	expectedFiles := []string{
		"nested-dir/excluded-file.txt",
	}
	expectedFiles = append(expectedFiles, env.Assets.FilesInFolder()...)
	env.Assets.ValidateFilesAreEqual(bundleDir, outDir, expectedFiles)
}

func TestBundleLockFile(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)

	bundleLockFilepath := filepath.Join(env.Assets.CreateTempFolder("bundle-lock"), "imgpkg-bundle-lock-test.yml")

	// push the bundle in the assets dir
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir, "--lock-output", bundleLockFilepath})

	bundleBs, err := os.ReadFile(bundleLockFilepath)
	require.NoError(t, err)

	// Keys are written in alphabetical order
	expectedYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
bundle:
  image: %s@sha256:[A-Fa-f0-9]{64}
  tag: latest
kind: BundleLock
`, env.Image)

	require.Regexp(t, expectedYml, string(bundleBs))

	outputDir := env.Assets.CreateTempFolder("bundle-pull")
	imgpkg.Run([]string{"pull", "--lock", bundleLockFilepath, "-o", outputDir})

	env.Assets.ValidateFilesAreEqual(bundleDir, outputDir, env.Assets.FilesInFolder())
}

func TestImagePullOnBundleError(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir})

	var stderrBs bytes.Buffer

	path := env.Assets.CreateTempFolder("not-used")
	_, err := imgpkg.RunWithOpts([]string{"pull", "-i", env.Image, "-o", path},
		helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	require.Error(t, err)
	assert.Contains(t, errOut, "Expected bundle flag when pulling a bundle (hint: Use -b instead of -i for bundles)")
}

func TestBundlePullOnImageError(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	imageDir := env.Assets.CreateAndCopySimpleApp("image-folder")
	imgpkg.Run([]string{"push", "-i", env.Image, "-f", imageDir})

	path := env.Assets.CreateTempFolder("not-used")
	var stderrBs bytes.Buffer
	_, err := imgpkg.RunWithOpts([]string{"pull", "-b", env.Image, "-o", path},
		helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})

	require.Error(t, err)

	errOut := stderrBs.String()

	require.Contains(t, errOut, "Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
}
