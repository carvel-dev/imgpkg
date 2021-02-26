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
)

func TestCopyWithBundleLockInputWithIndexesToRepoDestinationAndOutputNewBundleLockFile(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	// create generic image
	imageLockYAML := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
 - annotations:
     kbld.carvel.dev/id: index.docker.io/library/nginx@sha256:4cf620a5c81390ee209398ecc18e5fb9dd0f5155cd82adcbae532fec94006fb9
   image: index.docker.io/library/nginx@sha256:4cf620a5c81390ee209398ecc18e5fb9dd0f5155cd82adcbae532fec94006fb9
`

	// create a bundle with ref to generic
	testDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

	// create bundle that refs image with --lock-output and a random tag based on time
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	imgpkg.Run([]string{"push", "-b", fmt.Sprintf("%s:%v", env.Image, time.Now().UnixNano()), "-f", testDir, "--lock-output", lockFile})

	// copy via output file
	lockOutputPath := filepath.Join(testDir, "bundle-lock-relocate-lock.yml")
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	// check if nginx Index image is present in the repository
	refs := []string{
		env.RelocationRepo + "@sha256:4cf620a5c81390ee209398ecc18e5fb9dd0f5155cd82adcbae532fec94006fb9",
	}
	if err := env.Assert.ValidateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyWithBundleLockInputToRepoDestinationAndOutputNewBundleLockFile(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	// create generic image

	imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
	imageDigestRef := env.Image + imageDigest

	imgLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imgLockYAML)

	// create bundle that refs image with --lock-ouput and a random tag based on time
	testDir := env.Assets.CreateTempFolder("copy-with-lock-file")
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	imgpkg.Run([]string{"push", "-b", fmt.Sprintf("%s:%v", env.Image, time.Now().UnixNano()), "-f", bundleDir, "--lock-output", lockFile})
	bundleLock, err := lockconfig.NewBundleLockFromPath(lockFile)
	if err != nil {
		t.Fatalf("failed to read bundlelock file: %v", err)
	}
	bundleDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, bundleLock.Bundle.Image))
	bundleTag := bundleLock.Bundle.Tag

	// copy via output file
	lockOutputPath := filepath.Join(testDir, "bundle-lock-relocate-lock.yml")
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
	env.Assert.AssertBundleLock(lockOutputPath, relocatedRef, bundleTag)

	// check if bundle and referenced images are present in dst repo
	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest, env.RelocationRepo + ":" + bundleTag}
	if err := env.Assert.ValidateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyWithImageLockInputToRepoDestinationAndOutputNewImageLockFile(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	// create generic image
	imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	testDir := env.Assets.CreateTempFolder("copy-image-to-repo-with-lock-file")
	lockFile := filepath.Join(testDir, "images.lock.yml")
	err := ioutil.WriteFile(lockFile, []byte(imageLockYAML), 0700)
	if err != nil {
		t.Fatalf("failed to create images.lock file: %v", err)
	}

	// copy via output file
	lockOutputPath := filepath.Join(testDir, "image-relocate-lock.yml")
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	env.Assert.AssertImagesLock(lockOutputPath, []lockconfig.ImageRef{{Image: expectedRef}})

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigest}
	if err := env.Assert.ValidateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyWithBundleLockInputToTarFileAndToADifferentRepoCheckTagIsKeptAndBundleLockFileIsGenerated(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	// create generic image
	imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	// create bundle that refs image with --lock-ouput
	testDir := env.Assets.CreateTempFolder("copy-bundle-via-tar-keep-tag")
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir, "--lock-output", lockFile})

	origBundleLock, err := lockconfig.NewBundleLockFromPath(lockFile)
	if err != nil {
		t.Fatalf("unable to read original bundle lock: %s", err)
	}

	bundleDigestRef := fmt.Sprintf("%s@%s", env.Image, helpers.ExtractDigest(t, origBundleLock.Bundle.Image))

	// copy via output file
	tarFilePath := filepath.Join(testDir, "bundle.tar")
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-tar", tarFilePath})

	env.Assert.ImagesDigestIsOnTar(tarFilePath, imageDigestRef, bundleDigestRef)

	// copy from tar to repo
	lockFilePath := filepath.Join(testDir, "relocate-from-tar-lock.yml")
	imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})

	expectedRelocatedRef := fmt.Sprintf("%s@%s", env.RelocationRepo, helpers.ExtractDigest(t, bundleDigestRef))
	env.Assert.AssertBundleLock(lockFilePath, expectedRelocatedRef, origBundleLock.Bundle.Tag)

	// validate bundle and image were relocated
	relocatedBundleRef := expectedRelocatedRef
	relocatedImageRef := env.RelocationRepo + imageDigest
	relocatedBundleTagRef := fmt.Sprintf("%s:%v", env.RelocationRepo, origBundleLock.Bundle.Tag)

	if err := env.Assert.ValidateImagesPresenceInRegistry([]string{relocatedBundleRef, relocatedImageRef, relocatedBundleTagRef}); err != nil {
		t.Fatalf("Failed to locate digest in relocationRepo: %v", err)
	}
}

func TestCopyWithImageLockInputToTarFileAndToADifferentRepoCheckTagIsKeptAndImageLockFileIsGenerated(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	testDir := env.Assets.CreateTempFolder("copy--image-lock-via-tar-keep-tag")
	lockFile := filepath.Join(testDir, "images.lock.yml")

	err := ioutil.WriteFile(lockFile, []byte(imageLockYAML), 0700)
	if err != nil {
		t.Fatalf("failed to create images.lock file: %v", err)
	}

	// copy via output file
	tarFilePath := filepath.Join(testDir, "image.tar")
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-tar", tarFilePath})

	env.Assert.ImagesDigestIsOnTar(tarFilePath, imageDigestRef)

	// copy from tar to repo
	lockOutputPath := filepath.Join(testDir, "relocate-from-tar-lock.yml")
	imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	env.Assert.AssertImagesLock(lockOutputPath, []lockconfig.ImageRef{{Image: expectedRef}})

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigest}
	if err := env.Assert.ValidateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}
