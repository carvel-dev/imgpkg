// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildBundleOfBundles(t *testing.T) {
	env := helpers.BuildEnv(t)
	logger := helpers.Logger{}
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	bundleDigestRef := ""
	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
	logger.Section("create inner bundle", func() {
		out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
		bundleDigestRef = fmt.Sprintf("%s@%s", env.Image, helpers.ExtractDigest(t, out))
	})

	tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
	tempBundleTarFile := filepath.Join(tempBundleTarDir, "bundle-tar.tgz")

	logger.Section("create new bundle with bundles", func() {
		imagesLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, bundleDigestRef)
		env.BundleFactory.AddFileToBundle(filepath.Join(".imgpkg", "images.yml"), imagesLockYAML)

		out := imgpkg.Run([]string{"build", "--tty", "-b", env.Image, "-f", bundleDir, "--to-tar", tempBundleTarFile})
		indexOfImageDigest := strings.Index(out, fmt.Sprintf("Built '%s", env.Image))
		assertTarballLabelsOuterBundle(tempBundleTarFile, fmt.Sprintf("%s@%s", env.Image, helpers.ExtractDigest(t, out[indexOfImageDigest:])), t)
	})
}

func TestBuildFilesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test as this is a known issue: https://github.com/vmware-tanzu/carvel-imgpkg/issues/270")
	}

	env := helpers.BuildEnv(t)
	logger := helpers.Logger{}
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	// We need this chmod, because in the github action this file permission is converted into
	// u+rw even if in the this repository the permission is correct
	require.NoError(t, os.Chmod(filepath.Join(".", "assets", "bundle_file_permissions", "read_only_config.yml"), 0400))

	tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
	tempBundleTarFile := filepath.Join(tempBundleTarDir, "bundle-tar.tgz")

	logger.Section("Build bundle with different permissions files", func() {
		imgpkg.Run([]string{"build", "-f", "./assets/bundle_file_permissions", "-b", env.Image, "--to-tar", tempBundleTarFile})
	})

	logger.Section("Copy locally built bundle into registry", func() {
		imgpkg.Run([]string{"copy", "--tar", tempBundleTarFile, "--to-repo", env.Image})
	})

	bundleDir := env.Assets.CreateTempFolder("bundle-location")

	logger.Section("Pull bundle", func() {
		imgpkg.Run([]string{"pull", "-b", env.Image, "-o", bundleDir})
	})

	logger.Section("Check files permissions did not change", func() {
		info, err := os.Stat(filepath.Join(bundleDir, "exec_file.sh"))
		require.NoError(t, err)
		assert.Equal(t, fs.FileMode(0700).String(), info.Mode().String(), "have -rwx------ permissions")
		info, err = os.Stat(filepath.Join(bundleDir, "read_only_config.yml"))
		require.NoError(t, err)
		assert.Equal(t, fs.FileMode(0400).String(), info.Mode().String(), "have -r-------- permissions")
		info, err = os.Stat(filepath.Join(bundleDir, "read_write_config.yml"))
		require.NoError(t, err)
		assert.Equal(t, fs.FileMode(0600).String(), info.Mode().String(), "have -rw------- permissions")
	})
}

func TestBuildBundlePullAnnotation(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
	tempBundleTarFile := filepath.Join(tempBundleTarDir, "bundle-tar.tgz")

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
	imgpkg.Run([]string{"build", "-b", env.Image, "-f", bundleDir, "--to-tar", tempBundleTarFile})
	imgpkg.Run([]string{"copy", "--to-repo", env.Image, "--tar", tempBundleTarFile})

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

func TestBuildWithFileExclusion(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)

	env.BundleFactory.AddFileToBundle("excluded-file.txt", "I will not be present in the bundle")
	env.BundleFactory.AddFileToBundle(
		filepath.Join("nested-dir", "excluded-file.txt"),
		"this file will not be excluded because it is nested",
	)
	tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
	tempBundleTarFile := filepath.Join(tempBundleTarDir, "bundle-tar.tgz")

	imgpkg.Run([]string{"build", "-b", env.Image, "-f", bundleDir, "--file-exclusion", "excluded-file.txt", "--to-tar", tempBundleTarFile})
	imgpkg.Run([]string{"copy", "--to-repo", env.Image, "--tar", tempBundleTarFile})

	outDir := env.Assets.CreateTempFolder("bundle-exclusion")
	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", outDir})

	expectedFiles := []string{
		"nested-dir/excluded-file.txt",
	}
	expectedFiles = append(expectedFiles, env.Assets.FilesInFolder()...)
	env.Assets.ValidateFilesAreEqual(bundleDir, outDir, expectedFiles)
}

func TestBuildImage(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	testDir := env.Assets.CreateTempFolder("imgpkg-test-basic")

	tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
	tempBundleTarFile := filepath.Join(tempBundleTarDir, "bundle-tar.tgz")

	imgpkg.Run([]string{"build", "-i", env.Image, "-f", env.Assets.SimpleAppDir(), "--to-tar", tempBundleTarFile})
	imgpkg.Run([]string{"copy", "--tar", tempBundleTarFile, "--to-repo", env.Image})
	imgpkg.Run([]string{"pull", "-i", env.Image, "-o", testDir})

	env.Assets.ValidateFilesAreEqual(env.Assets.SimpleAppDir(), testDir, []string{
		"README.md",
		"LICENSE",
		"config/config.yml",
		"config/inner-dir/README.txt",
	})
}

func TestBuildWithTagIsPreserved(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	tempBundleTarDir := env.Assets.CreateTempFolder("bundle-tar")
	tempBundleTarFile := filepath.Join(tempBundleTarDir, "bundle-tar.tgz")

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
	bundleRefWithTag := env.Image + ":tag1"
	imgpkg.Run([]string{"build", "-b", bundleRefWithTag, "-f", bundleDir, "--to-tar", tempBundleTarFile})
	imgpkg.Run([]string{"copy", "--to-repo", env.Image, "--tar", tempBundleTarFile})

	ref, _ := name.NewTag(bundleRefWithTag, name.WeakValidation)
	_, err := remote.Head(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(t, err)
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
