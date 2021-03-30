// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyBundleToDifferentRepository(t *testing.T) {
	logger := helpers.Logger{}

	t.Run("when all images are collocated it copies all images from the original location and generate a BundleLock", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
		var bundleDigest, imageDigest string
		logger.Section("create bundle with image", func() {
			imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
`, env.Image, imageDigest)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s%s", env.Image, bundleTag), "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to repository and generate BundleLock", func() {
			lockOutputPath := filepath.Join(env.Assets.CreateTempFolder("bundle-lock"), "bundle-relocate-lock.yml")
			imgpkg.Run([]string{"copy",
				"--bundle", fmt.Sprintf("%s%s", env.Image, bundleTag),
				"--to-repo", env.RelocationRepo,
				"--lock-output", lockOutputPath},
			)

			expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
			expectedTag := strings.TrimPrefix(bundleTag, ":")
			env.Assert.AssertBundleLock(lockOutputPath, expectedRef, expectedTag)
		})

		refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleTag, env.RelocationRepo + bundleDigest}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
	})

	t.Run("when some images are not in the same repository as the bundle it copies all images", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()

		image := env.Image + "-image-outside-repo"
		var imageDigest, bundleDigestRef, bundleDigest string
		logger.Section("create bundle with image", func() {
			imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, image)
			// image intentionally does not exist in bundle repo
			imageDigestRef := image + imageDigest

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			bundleDigestRef = env.Image + bundleDigest
		})

		imgpkg.Run([]string{"copy", "--bundle", bundleDigestRef, "--to-repo", env.RelocationRepo})

		refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
	})

	t.Run("when copying bundle with annotations in ImagesLock it maintain the annotation after the copy", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()

		var bundleDigestRef, bundleDigest string
		logger.Section("create bundle with image", func() {
			imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
			imageDigestRef := env.Image + imageDigest

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
  annotations:
    greeting: hello world
`, imageDigestRef)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			bundleDigestRef = env.Image + bundleDigest
		})

		logger.Section("copy bundle", func() {
			imgpkg.Run([]string{"copy", "-b", bundleDigestRef, "--to-repo", env.RelocationRepo})
		})

		testDir := env.Assets.CreateTempFolder("test-annotation")
		logger.Section("pull bundle to read the ImagesLock file", func() {
			bundleDigestRef = env.RelocationRepo + bundleDigest
			imgpkg.Run([]string{"pull", "-b", bundleDigestRef, "-o", testDir})
		})

		imgLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(testDir, ".imgpkg", "images.yml"))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"greeting": "hello world"}, imgLock.Images[0].Annotations)
	})

	t.Run("when bundle contains a bundle it copies all the bundle's images from the original location", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest string
		logger.Section("create 2 simple images", func() {

			img1DigestRef = imgRef.Context().Name() + "-img1"
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img1DigestRef)
			img1DigestRef = img1DigestRef + img1Digest

			img2DigestRef = imgRef.Context().Name() + "-img2"
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img2DigestRef)
			img2DigestRef = img2DigestRef + img2Digest
		})

		simpleBundle := imgRef.Context().Name() + "-simple-bundle"
		simpleBundleDigest := ""
		logger.Section("create simple bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", simpleBundle, "-f", bundleDir})
			simpleBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		nestedBundle := imgRef.Context().Name() + "-bundle-nested"
		nestedBundleDigest := ""
		logger.Section("create nested bundle that contains images and the simple bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
- image: %s
`, img1DigestRef, img2DigestRef, simpleBundle+simpleBundleDigest)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", nestedBundle, "-f", bundleDir})
			nestedBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		outerBundle := imgRef.Context().Name() + "-bundle-outer"
		outerBundleDigest := ""
		bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
		bundleToCopy := fmt.Sprintf("%s%s", outerBundle, bundleTag)
		logger.Section("create outer bundle with image, simple bundle and nested bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
- image: %s
`, nestedBundle+nestedBundleDigest, img1DigestRef, simpleBundle+simpleBundleDigest)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", bundleToCopy, "-f", bundleDir})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to repository", func() {
			imgpkg.Run([]string{"copy",
				"--bundle", bundleToCopy,
				"--to-repo", env.RelocationRepo,
			},
			)
		})

		refs := []string{
			env.RelocationRepo + img1Digest,
			env.RelocationRepo + img2Digest,
			env.RelocationRepo + simpleBundleDigest,
			env.RelocationRepo + nestedBundleDigest,
			env.RelocationRepo + bundleTag,
			env.RelocationRepo + outerBundleDigest,
		}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
	})
}

func TestCopyBundleUsingTar(t *testing.T) {
	logger := helpers.Logger{}
	t.Run("when a bundle contains other bundles it copies all images from all bundles", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("nested-bundles-tar-test")
		tarFilePath := filepath.Join(testDir, "bundle.tar")

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest string
		logger.Section("create 2 simple images", func() {
			img1DigestRef = imgRef.Context().Name() + "-img1"
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img1DigestRef)
			img1DigestRef = img1DigestRef + img1Digest

			img2DigestRef = imgRef.Context().Name() + "-img2"
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img2DigestRef)
			img2DigestRef = img2DigestRef + img2Digest
		})

		nestedBundle := imgRef.Context().Name() + "-bundle-nested"
		nestedBundleDigest := ""
		logger.Section("create nested bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
`, img1DigestRef, img2DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", nestedBundle, "-f", bundleDir})
			nestedBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		outerBundle := imgRef.Context().Name() + "-bundle-outer"
		outerBundleDigest := ""
		logger.Section("create outer bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
`, nestedBundle+nestedBundleDigest, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", outerBundle, "-f", bundleDir})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("export full bundle to tar", func() {
			imgpkg.Run([]string{"copy", "-b", outerBundle + outerBundleDigest, "--to-tar", tarFilePath})
		})

		lockFilePath := filepath.Join(testDir, "relocate-from-tar-lock.yml")
		logger.Section("import bundle to new repository", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})
			relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, outerBundleDigest)
			env.Assert.AssertBundleLock(lockFilePath, relocatedRef, "")
		})

		imagesToCheck := []string{
			env.RelocationRepo + outerBundleDigest,
			env.RelocationRepo + nestedBundleDigest,
			env.RelocationRepo + img1Digest,
			env.RelocationRepo + img2Digest,
		}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(imagesToCheck))
	})

	t.Run("when bundle contains only images it copies all images", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("tar-test")
		tarFilePath := filepath.Join(testDir, "bundle.tar")

		var bundleDigest, imageDigest string
		logger.Section("create bundle with image", func() {
			// create generic image
			imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
			imageDigestRef := env.Image + imageDigest

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy images to a tar file", func() {
			imgpkg.Run([]string{"copy", "-b", env.Image, "--to-tar", tarFilePath})
		})

		lockFilePath := filepath.Join(testDir, "relocate-from-tar-lock.yml")

		logger.Section("import images to the new registry", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})
			relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
			env.Assert.AssertBundleLock(lockFilePath, relocatedRef, "latest")
		})

		imagesToCheck := []string{
			env.RelocationRepo + bundleDigest,
			env.RelocationRepo + imageDigest,
		}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(imagesToCheck))
	})

	t.Run("when bundle have tag it preserves the tag in the destination", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("tar-tag-test")
		tarFilePath := filepath.Join(testDir, "bundle.tar")

		var tag int64
		logger.Section("create bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
`)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			tag = time.Now().UnixNano()
			imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s:%v", env.Image, tag), "-f", bundleDir})
		})

		logger.Section("copy images to a tar file", func() {
			imgpkg.Run([]string{"copy", "-b", fmt.Sprintf("%s:%v", env.Image, tag), "--to-tar", tarFilePath})
		})

		logger.Section("import images to the new registry", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo})
		})

		imagesToCheck := []string{
			fmt.Sprintf("%s:%v", env.RelocationRepo, tag),
		}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(imagesToCheck))
	})
}

func TestCopyErrorsWhenCopyBundleUsingImageFlag(t *testing.T) {
	logger := helpers.Logger{}
	t.Run("when trying to copy a bundle using the -i flag, it fails", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()

		bundleDigestRef := ""
		logger.Section("create bundle with image", func() {
			imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
			imageDigestRef := env.Image + imageDigest

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			bundleDigestRef = env.Image + bundleDigest
		})

		var stderrBs bytes.Buffer
		_, err := imgpkg.RunWithOpts([]string{"copy", "-i", bundleDigestRef, "--to-tar", "fake_path"},
			helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})
		errOut := stderrBs.String()

		require.Error(t, err)
		assert.Contains(t, errOut, "Expected bundle flag when copying a bundle (hint: Use -b instead of -i for bundles)")
	})
}
