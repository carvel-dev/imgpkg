// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/require"
)

func TestCopyFromBundleLock(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
	logger := helpers.Logger{}
	defer env.Cleanup()

	randomImageDigest := ""
	randomImageDigestRef := ""
	logger.Section("create random image for tests", func() {
		randomImageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
		randomImageDigestRef = env.Image + randomImageDigest
	})

	t.Run("when copying index to repo, it is successful and generates a BundleLock file", func(t *testing.T) {
		testDir := ""
		lockFile := ""
		logger.Section("create bundle from index", func() {
			imageLockYAML := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
 - annotations:
     kbld.carvel.dev/id: index.docker.io/library/nginx@sha256:4cf620a5c81390ee209398ecc18e5fb9dd0f5155cd82adcbae532fec94006fb9
   image: index.docker.io/library/nginx@sha256:4cf620a5c81390ee209398ecc18e5fb9dd0f5155cd82adcbae532fec94006fb9
`
			testDir = env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			lockFile = filepath.Join(testDir, "bundle.lock.yml")
			imgpkg.Run([]string{"push", "-b", fmt.Sprintf("%s:%v", env.Image, time.Now().UnixNano()), "-f", testDir, "--lock-output", lockFile})
		})

		logger.Section("copy using the BundleLock to a repository", func() {
			lockOutputPath := filepath.Join(testDir, "bundle-lock-relocate-lock.yml")
			imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})
		})

		logger.Section("validate the index is present in the destination", func() {
			refs := []string{
				env.RelocationRepo + "@sha256:4cf620a5c81390ee209398ecc18e5fb9dd0f5155cd82adcbae532fec94006fb9",
			}
			require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs), "validating image presence")
		})
	})

	t.Run("when Copying images to Repo, it is successful and generates a BundleLock file", func(t *testing.T) {
		lockFile := ""
		testDir := ""
		logger.Section("create bundle", func() {
			imgLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, randomImageDigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imgLockYAML)

			testDir = env.Assets.CreateTempFolder("copy-with-lock-file")
			lockFile = filepath.Join(testDir, "bundle.lock.yml")
			imgpkg.Run([]string{"push", "-b", fmt.Sprintf("%s:%v", env.Image, time.Now().UnixNano()), "-f", bundleDir, "--lock-output", lockFile})
		})

		bundleLock, err := lockconfig.NewBundleLockFromPath(lockFile)
		require.NoError(t, err)

		bundleDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, bundleLock.Bundle.Image))
		bundleTag := bundleLock.Bundle.Tag

		lockOutputPath := filepath.Join(testDir, "bundle-lock-relocate-lock.yml")
		logger.Section("copy bundle using the lock file", func() {
			imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})
		})

		relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
		env.Assert.AssertBundleLock(lockOutputPath, relocatedRef, bundleTag)

		refs := []string{env.RelocationRepo + randomImageDigest, env.RelocationRepo + bundleDigest, env.RelocationRepo + ":" + bundleTag}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs), "validating image presence")
	})

	t.Run("when Copying images to Tar file and after importing to a new Repo, it keeps the bundle tag and generates a BundleLock file", func(t *testing.T) {
		testDir := env.Assets.CreateTempFolder("copy-bundle-via-tar-keep-tag")
		lockFile := filepath.Join(testDir, "bundle.lock.yml")

		logger.Section("create bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, randomImageDigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir, "--lock-output", lockFile})
		})

		origBundleLock, err := lockconfig.NewBundleLockFromPath(lockFile)
		require.NoError(t, err)

		bundleDigestRef := fmt.Sprintf("%s@%s", env.Image, helpers.ExtractDigest(t, origBundleLock.Bundle.Image))

		tarFilePath := filepath.Join(testDir, "bundle.tar")
		logger.Section("copy bundle to tar", func() {
			imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-tar", tarFilePath})

			env.Assert.ImagesDigestIsOnTar(tarFilePath, randomImageDigestRef, bundleDigestRef)
		})

		logger.Section("import tar to a new repository", func() {
			lockFilePath := filepath.Join(testDir, "relocate-from-tar-lock.yml")
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})

			expectedRelocatedRef := fmt.Sprintf("%s@%s", env.RelocationRepo, helpers.ExtractDigest(t, bundleDigestRef))
			env.Assert.AssertBundleLock(lockFilePath, expectedRelocatedRef, origBundleLock.Bundle.Tag)

			relocatedBundleRef := expectedRelocatedRef
			relocatedImageRef := env.RelocationRepo + randomImageDigest
			relocatedBundleTagRef := fmt.Sprintf("%s:%v", env.RelocationRepo, origBundleLock.Bundle.Tag)

			require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry([]string{relocatedBundleRef, relocatedImageRef, relocatedBundleTagRef}))
		})
	})

	t.Run("when Copying bundle that contains a bundle it is successful", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}

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
		var lockFile string

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
			lockFile = filepath.Join(bundleDir, "bundle.lock.yml")
			out := imgpkg.Run([]string{"push", "--tty", "-b", bundleToCopy, "-f", bundleDir, "--lock-output", lockFile})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to repository", func() {
			imgpkg.Run([]string{"copy",
				"--lock", lockFile,
				"--to-repo", env.RelocationRepo},
			)
		})

		logger.Section("validate the index is present in the destination", func() {
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
	})

	t.Run("When Copying bundle it generates image with locations", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
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
		var lockFile string

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
			lockFile = filepath.Join(bundleDir, "bundle.lock.yml")
			out := imgpkg.Run([]string{"push", "--tty", "-b", bundleToCopy, "-f", bundleDir, "--lock-output", lockFile})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to repository", func() {
			out := imgpkg.Run([]string{"copy",
				"--lock", lockFile,
				"--to-repo", env.RelocationRepo},
			)
			fmt.Println(out)
		})

		logger.Section("download the locations file for outer bundle and check it", func() {
			downloadAndCheckLocationsFile(t, env, outerBundleDigest[1:], bundle.ImageLocationsConfig{
				APIVersion: "imgpkg.carvel.dev/v1alpha1",
				Kind:       "ImageLocations",
				Images: []bundle.ImageLocation{
					{
						Image:    nestedBundle + nestedBundleDigest,
						IsBundle: true,
					},
					{
						Image:    img1DigestRef,
						IsBundle: false,
					},
					{
						Image:    simpleBundle + simpleBundleDigest,
						IsBundle: true,
					},
				},
			})
		})

		logger.Section("download the locations file for nested bundle and check it", func() {
			downloadAndCheckLocationsFile(t, env, nestedBundleDigest[1:], bundle.ImageLocationsConfig{
				APIVersion: "imgpkg.carvel.dev/v1alpha1",
				Kind:       "ImageLocations",
				Images: []bundle.ImageLocation{
					{
						Image:    img1DigestRef,
						IsBundle: false,
					},
					{
						Image:    img2DigestRef,
						IsBundle: false,
					},
					{
						Image:    simpleBundle + simpleBundleDigest,
						IsBundle: true,
					},
				},
			})
		})

		logger.Section("download the locations file for simple bundle and check it", func() {
			downloadAndCheckLocationsFile(t, env, simpleBundleDigest[1:], bundle.ImageLocationsConfig{
				APIVersion: "imgpkg.carvel.dev/v1alpha1",
				Kind:       "ImageLocations",
				Images: []bundle.ImageLocation{
					{
						Image:    img1DigestRef,
						IsBundle: false,
					},
				},
			})
		})
	})
}

func TestCopyFromImageLock(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	logger := helpers.Logger{}
	defer env.Cleanup()

	randomImageDigest := ""
	randomImageDigestRef := ""
	logger.Section("create random image for tests", func() {
		randomImageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
		randomImageDigestRef = env.Image + randomImageDigest
	})

	t.Run("when copying to repo, it is successful and generates an ImageLock file", func(t *testing.T) {
		env.UpdateT(t)
		imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
  annotations:
    some-annotation: some-value
`, randomImageDigestRef)

		testDir := env.Assets.CreateTempFolder("copy-image-to-repo-with-lock-file")
		lockFile := filepath.Join(testDir, "images.lock.yml")
		err := ioutil.WriteFile(lockFile, []byte(imageLockYAML), 0700)
		require.NoError(t, err)

		logger.Section("copy from lock file", func() {
			lockOutputPath := filepath.Join(testDir, "image-relocate-lock.yml")
			imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

			imageRefs := []lockconfig.ImageRef{{
				Image:       fmt.Sprintf("%s%s", env.RelocationRepo, randomImageDigest),
				Annotations: map[string]string{"some-annotation": "some-value"},
			}}
			env.Assert.AssertImagesLock(lockOutputPath, imageRefs)

			refs := []string{env.RelocationRepo + randomImageDigest}
			require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
		})
	})

	t.Run("when Copying images to Tar file and after importing to a new Repo, it keeps the tags and generates a ImageLock file", func(t *testing.T) {
		env.UpdateT(t)
		imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, randomImageDigestRef)

		testDir := env.Assets.CreateTempFolder("copy--image-lock-via-tar-keep-tag")
		lockFile := filepath.Join(testDir, "images.lock.yml")

		err := ioutil.WriteFile(lockFile, []byte(imageLockYAML), 0700)
		require.NoError(t, err)

		tarFilePath := filepath.Join(testDir, "image.tar")
		logger.Section("copy image to tar file", func() {
			imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-tar", tarFilePath})

			env.Assert.ImagesDigestIsOnTar(tarFilePath, randomImageDigestRef)
		})

		lockOutputPath := filepath.Join(testDir, "relocate-from-tar-lock.yml")
		logger.Section("import tar to new repository", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

			expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, randomImageDigest)
			env.Assert.AssertImagesLock(lockOutputPath, []lockconfig.ImageRef{{Image: expectedRef}})

			refs := []string{env.RelocationRepo + randomImageDigest}
			require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
		})
	})
}
