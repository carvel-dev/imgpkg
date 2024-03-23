// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"carvel.dev/imgpkg/pkg/imgpkg/bundle"
	"carvel.dev/imgpkg/pkg/imgpkg/lockconfig"
	"carvel.dev/imgpkg/test/helpers"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullImageLockRewrite(t *testing.T) {
	logger := &helpers.Logger{}

	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	imageDigestRef := "@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6"
	dockerhubImgRef := helpers.CompleteImageRef("library/hello-world")
	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
`, dockerhubImgRef, imageDigestRef)

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

	out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
	bundleDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))

	imgpkg.Run([]string{"copy", "-b", env.Image, "--to-repo", env.Image})

	pullDir := env.Assets.CreateTempFolder("pull-rewrite-lock")
	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", pullDir})

	expectedImageRef := env.Image + imageDigestRef
	env.Assert.AssertImagesLock(filepath.Join(pullDir, ".imgpkg", "images.yml"), []lockconfig.ImageRef{{Image: expectedImageRef}})

	hash, err := v1.NewHash(bundleDigest[1:])
	require.NoError(t, err)
	locationImg := fmt.Sprintf("%s:%s-%s.image-locations.imgpkg", env.Image, hash.Algorithm, hash.Hex)

	logger.Section("download the locations file and check it", func() {
		locationImgFolder := env.Assets.CreateTempFolder("locations-img")
		env.ImageFactory.Download(locationImg, locationImgFolder)

		locationsFilePath := filepath.Join(locationImgFolder, "image-locations.yml")
		require.FileExists(t, locationsFilePath)

		cfg, err := bundle.NewLocationConfigFromPath(locationsFilePath)
		require.NoError(t, err)

		require.Equal(t, bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images: []bundle.ImageLocation{{
				Image: dockerhubImgRef + imageDigestRef,
				// Repository not used for now because all images will be present in the same repository
				IsBundle: false,
			}},
		}, cfg)
	})

}

func TestPullImageLockRewriteBundleOfBundles(t *testing.T) {
	env := helpers.BuildEnv(t)
	logger := helpers.Logger{}
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	bundleDigestRef := ""
	imageDigestRef := "@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6"
	dockerhubImgRef := helpers.CompleteImageRef("library/hello-world")
	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
`, dockerhubImgRef, imageDigestRef)

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

		outerBundleOut := imgpkg.Run([]string{"push", "--tty", "-b", uniqueImageName, "-f", bundleDir})
		outerBundleDigestRef := helpers.ExtractDigest(t, outerBundleOut)

		imgpkg.Run([]string{"copy", "-b", uniqueImageName + "@" + outerBundleDigestRef, "--to-repo", uniqueImageName})

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
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	bundleDigestRef := ""
	logger.Section("create inner bundle", func() {
		bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
		out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
		bundleDigestRef = helpers.ExtractDigest(t, out)
	})

	innerBundleRef := fmt.Sprintf("%s@%s", env.Image, bundleDigestRef)

	outerBundleDigest := ""
	outerBundle := env.RelocationRepo
	outerBundleTag := fmt.Sprintf("some-tag-%d", time.Now().Nanosecond())
	logger.Section("create new bundle with bundles", func() {
		bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
		imagesLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, innerBundleRef)
		env.BundleFactory.AddFileToBundle(filepath.Join(".imgpkg", "images.yml"), imagesLockYAML)

		out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s:%s", outerBundle, outerBundleTag), "-f", bundleDir})
		outerBundleDigest = helpers.ExtractDigest(t, out)
	})

	outerBundleRef := fmt.Sprintf("%s@%s", outerBundle, outerBundleDigest)

	t.Run("pull bundle recursively and downloads the ImagesLock for all nested bundles", func(t *testing.T) {
		outDir := env.Assets.CreateTempFolder("bundle-annotation")

		imgpkg.Run([]string{"pull", "--recursive", "-b", outerBundleRef, "-o", outDir})

		subBundleDirectoryPath := strings.ReplaceAll(bundleDigestRef, "sha256:", "sha256-")
		assert.DirExists(t, filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath))
		assert.FileExists(t, filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath, ".imgpkg", "images.yml"))
		assert.FileExists(t, filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath, ".imgpkg", "bundle.yml"))

		innerBundleImagesYmlContent, err := os.ReadFile(filepath.Join(outDir, ".imgpkg", "bundles", subBundleDirectoryPath, ".imgpkg", "images.yml"))
		assert.NoError(t, err)
		assert.Equal(t, helpers.ImagesYAML, string(innerBundleImagesYmlContent))
	})
}

func TestPullImageFromSlowServerShouldTimeout(t *testing.T) {
	logger := &helpers.Logger{}

	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	registry := helpers.NewFakeRegistry(t, logger)
	image := registry.WithRandomImage("random-image")
	registry.Build()
	defer registry.ResetHandler()

	registry.WithCustomHandler(func(writer http.ResponseWriter, request *http.Request) bool {
		time.Sleep(5 * time.Second)
		return false
	})

	actualErrOut := bytes.NewBufferString("")
	outDir := env.Assets.CreateTempFolder("bundle-annotation")
	imgpkg.RunWithOpts([]string{"pull", "--registry-response-header-timeout", "1s", "-i", image.RefDigest, "-o", outDir}, helpers.RunOpts{
		AllowError:   true,
		StdoutWriter: actualErrOut,
		StderrWriter: actualErrOut,
	})

	assert.Contains(t, actualErrOut.String(), "timeout awaiting response headers")
}

func TestPullImageIndexShouldError(t *testing.T) {
	logger := &helpers.Logger{}

	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	registry := helpers.NewFakeRegistry(t, logger)
	imageIndex := registry.WithARandomImageIndex("random-image-index", 3)
	registry.Build()
	defer registry.ResetHandler()

	pullDir := env.Assets.CreateTempFolder("pull-rewrite-lock")
	out := bytes.NewBufferString("")
	_, err := imgpkg.RunWithOpts([]string{"pull", "--tty", "-i", imageIndex.RefDigest, "-o", pullDir}, helpers.RunOpts{
		AllowError:   true,
		StderrWriter: out,
		StdoutWriter: out,
	})

	assert.Error(t, err)
	assert.Contains(t, out.String(), "Unable to pull non-images, such as image indexes. (hint: provide a specific digest to the image instead)")
}

func TestPullImageOfBundle(t *testing.T) {
	logger := &helpers.Logger{}

	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	registry := helpers.NewFakeRegistry(t, logger)
	randomBundle := registry.WithBundleFromPath("repo/some-bundle-name", "assets/bundle")
	registry.Build()
	defer registry.CleanUp()

	t.Run("when --image-is-bundle-check is NOT provided it fails", func(t *testing.T) {
		pullDir := env.Assets.CreateTempFolder("unused-pull-bundle-image")
		out := bytes.NewBufferString("")
		_, err := imgpkg.RunWithOpts([]string{"pull", "--tty", "-i", randomBundle.RefDigest, "-o", pullDir}, helpers.RunOpts{
			AllowError:   true,
			StderrWriter: out,
			StdoutWriter: out,
		})

		require.Error(t, err)
		assert.Contains(t, out.String(), "Expected bundle flag when pulling a bundle (hint: Use -b instead of -i for bundles)")
	})

	t.Run("when --image-is-bundle-check=false is provided while using the -b flag it fails", func(t *testing.T) {
		pullDir := env.Assets.CreateTempFolder("pull-bundle-image")
		out := bytes.NewBufferString("")
		_, err := imgpkg.RunWithOpts([]string{"pull", "--tty", "-b", randomBundle.RefDigest, "-o", pullDir, "--image-is-bundle-check=false"}, helpers.RunOpts{
			AllowError:   true,
			StderrWriter: out,
			StdoutWriter: out,
		})

		require.Error(t, err)
		assert.Contains(t, out.String(), "Cannot set --image-is-bundle-check while using -b flag")
	})

	t.Run("when --image-is-bundle-check=false is provided downloads the OCI Image of the bundle", func(t *testing.T) {
		pullDir := env.Assets.CreateTempFolder("pull-bundle-image")
		imgpkg.RunWithOpts([]string{"pull", "--tty", "-i", randomBundle.RefDigest, "-o", pullDir, "--image-is-bundle-check=false"}, helpers.RunOpts{})
	})
}
