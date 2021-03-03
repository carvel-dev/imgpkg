// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

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
}

func TestCopyFromImageLock(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
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
