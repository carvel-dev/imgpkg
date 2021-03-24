// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
)

func TestPullImageLockRewrite(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	imageDigestRef := "@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6"
	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: hello-world%s
`, imageDigestRef)

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir})
	imgpkg.Run([]string{"copy", "-b", env.Image, "--to-repo", env.Image})

	pullDir := env.Assets.CreateTempFolder("pull-rewrite-lock")
	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", pullDir})

	expectedImageRef := env.Image + imageDigestRef
	env.Assert.AssertImagesLock(filepath.Join(pullDir, ".imgpkg", "images.yml"), []lockconfig.ImageRef{{Image: expectedImageRef}})
}

func TestPullImageLockRewriteBundleOfBundles(t *testing.T) {
	env := helpers.BuildEnv(t)
	logger := helpers.Logger{}
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	bundleDigestRef := ""
	imageDigestRef := "@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6"
	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: hello-world%s
`, imageDigestRef)

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
	uniqueImageName := env.Image + fmt.Sprintf("%d", time.Now().Unix())
	logger.Section("create inner bundle", func() {
		out := imgpkg.Run([]string{"push", "--tty", "-b", uniqueImageName, "-f", bundleDir})
		bundleDigestRef = helpers.ExtractDigest(t, out)
	})

	logger.Section("create new bundle with bundles", func() {
		imagesLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, fmt.Sprintf("%s@%s", uniqueImageName, bundleDigestRef))
		env.BundleFactory.AddFileToBundle(filepath.Join(".imgpkg", "images.yml"), imagesLockYAML)

		outerBundleOut := imgpkg.Run([]string{"push", "--tty", "-b", uniqueImageName, "-f", bundleDir, "--experimental-recursive-bundle"})
		outerBundleDigestRef := helpers.ExtractDigest(t, outerBundleOut)

		imgpkg.Run([]string{"copy", "--experimental-recursive-bundle", "-b", uniqueImageName + "@" + outerBundleDigestRef, "--to-repo", uniqueImageName})

		outDir := env.Assets.CreateTempFolder("bundle-annotation")

		imgpkg.Run([]string{"pull", "--recursive", "-b", uniqueImageName, "-o", outDir})

		subBundleDirectoryPath := strings.ReplaceAll(bundleDigestRef, "sha256:", "sha256-")
		assert.DirExists(t, filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath))
		assert.FileExists(t, filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath, ".imgpkg", "images.yml"))
		assert.FileExists(t, filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath, ".imgpkg", "bundle.yml"))

		innerBundleImagesYmlContent, err := os.ReadFile(filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath, ".imgpkg", "images.yml"))
		assert.NoError(t, err)

		assert.Regexp(t, fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
images:
- image: %s
kind: ImagesLock
`, uniqueImageName+imageDigestRef), string(innerBundleImagesYmlContent))
	})
}

func TestPullBundleOfBundles(t *testing.T) {
	env := helpers.BuildEnv(t)
	logger := helpers.Logger{}
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	bundleDigestRef := ""
	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
	logger.Section("create inner bundle", func() {
		out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
		bundleDigestRef = helpers.ExtractDigest(t, out)
	})

	logger.Section("create new bundle with bundles", func() {
		imagesLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, fmt.Sprintf("%s@%s", env.Image, bundleDigestRef))
		env.BundleFactory.AddFileToBundle(filepath.Join(".imgpkg", "images.yml"), imagesLockYAML)

		imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir, "--experimental-recursive-bundle"})

		outDir := env.Assets.CreateTempFolder("bundle-annotation")

		imgpkg.Run([]string{"pull", "--recursive", "-b", env.Image, "-o", outDir})

		subBundleDirectoryPath := strings.ReplaceAll(bundleDigestRef, "sha256:", "sha256-")
		assert.DirExists(t, filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath))
		assert.FileExists(t, filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath, ".imgpkg", "images.yml"))
		assert.FileExists(t, filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath, ".imgpkg", "bundle.yml"))

		innerBundleImagesYmlContent, err := os.ReadFile(filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath, ".imgpkg", "images.yml"))
		assert.NoError(t, err)
		assert.Equal(t, helpers.ImagesYAML, string(innerBundleImagesYmlContent))
	})
}
