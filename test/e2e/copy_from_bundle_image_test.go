// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

func TestCopyBundleWithCollocatedReferencedImagesToRepoDestinationAndOutputBundleLockFile(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Assets.cleanCreatedFolders()

	imageDigest := env.ImageFactory.pushSimpleAppImageWithRandomFile(imgpkg, env.Image)

	// create a bundle with ref to generic
	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
`, env.Image, imageDigest)
	bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imageLockYAML)

	// create bundle that refs image and a random tag based on time
	bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
	out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s%s", env.Image, bundleTag), "-f", bundleDir})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(t, out))

	lockOutputPath := filepath.Join(env.Assets.createTempFolder("bundle-lock"), "bundle-relocate-lock.yml")
	// copy via created ref
	imgpkg.Run([]string{"copy",
		"--bundle", fmt.Sprintf("%s%s", env.Image, bundleTag),
		"--to-repo", env.RelocationRepo,
		"--lock-output", lockOutputPath},
	)

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
	expectedTag := strings.TrimPrefix(bundleTag, ":")
	if err := env.BundleFactory.assertBundleLock(lockOutputPath, expectedRef, expectedTag); err != nil {
		t.Fatalf("validating bundle lock: %s", err)
	}

	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleTag, env.RelocationRepo + bundleDigest}
	if err := validateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyBundleWithNonCollocatedReferencedImagesToRepoDestination(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	//defer env.Assets.cleanCreatedFolders()

	image := env.Image + "-image-outside-repo"
	imageDigest := env.ImageFactory.pushSimpleAppImageWithRandomFile(imgpkg, image)
	// image intentionally does not exist in bundle repo
	imageDigestRef := image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)
	bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imageLockYAML)

	out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(t, out))
	bundleDigestRef := env.Image + bundleDigest

	imgpkg.Run([]string{"copy", "--bundle", bundleDigestRef, "--to-repo", env.RelocationRepo})

	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest}
	if err := validateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyBundleToTarFileAndToADifferentRepoCheckTagIsKeptAndBundleLockFileIsGenerated(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Assets.cleanCreatedFolders()

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-bundle-tar")
	tarFilePath := filepath.Join(testDir, "bundle.tar")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image
	imageDigest := env.ImageFactory.pushSimpleAppImageWithRandomFile(imgpkg, env.Image)
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imageLockYAML)

	tag := time.Now().UnixNano()
	// create bundle that refs image
	out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s:%v", env.Image, tag), "-f", bundleDir})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(t, out))

	// copy to a tar
	imgpkg.Run([]string{"copy", "-b", fmt.Sprintf("%s:%v", env.Image, tag), "--to-tar", tarFilePath})

	lockFilePath := filepath.Join(os.TempDir(), "relocate-from-tar-lock.yml")
	defer os.Remove(lockFilePath)

	// copy from tar to repo
	imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})
	relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
	err = env.BundleFactory.assertBundleLock(lockFilePath, relocatedRef, fmt.Sprintf("%v", tag))
	if err != nil {
		t.Fatalf("invalid bundle lock file: %s", err)
	}

	// validate bundle and image were relocated
	relocatedBundleRef := env.RelocationRepo + bundleDigest
	relocatedImageRef := env.RelocationRepo + imageDigest
	relocatedBundleTagRef := fmt.Sprintf("%s:%v", env.RelocationRepo, tag)

	if err := validateImagesPresenceInRegistry([]string{relocatedBundleRef, relocatedImageRef, relocatedBundleTagRef}); err != nil {
		t.Fatalf("Failed to locate digest in relocationRepo: %v", err)
	}
}

func TestCopyErrorsWhenCopyBundleUsingImageFlag(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Assets.cleanCreatedFolders()

	// create generic image
	imageDigest := env.ImageFactory.pushSimpleAppImageWithRandomFile(imgpkg, env.Image)
	imageDigestRef := env.Image + imageDigest

	// create a bundle with ref to generic
	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)
	bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imageLockYAML)

	out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(t, out))
	bundleDigestRef := env.Image + bundleDigest

	var stderrBs bytes.Buffer
	_, err := imgpkg.RunWithOpts([]string{"copy", "-i", bundleDigestRef, "--to-tar", "fake_path"},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	if err == nil {
		t.Fatalf("Expected incorrect flag error")
	}
	if !strings.Contains(errOut, "Expected bundle flag when copying a bundle (hint: Use -b instead of -i for bundles)") {
		t.Fatalf("Expected error to contain message about using the wrong copy flag, got: %s", errOut)
	}
}

func TestCopyBundlePreserveImgLockAnnotations(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Assets.cleanCreatedFolders()

	// create generic image
	imageDigest := env.ImageFactory.pushSimpleAppImageWithRandomFile(imgpkg, env.Image)
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
  annotations:
    greeting: hello world
`, imageDigestRef)
	bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imageLockYAML)

	// create bundle that refs image with --lock-ouput and a random tag based on time
	out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(t, out))
	bundleDigestRef := env.Image + bundleDigest

	// copy
	imgpkg.Run([]string{"copy", "-b", bundleDigestRef, "--to-repo", env.RelocationRepo})

	// pull
	testDir := env.Assets.createTempFolder("test-annotation")
	bundleDigestRef = env.RelocationRepo + bundleDigest
	imgpkg.Run([]string{"pull", "-b", bundleDigestRef, "-o", testDir})

	imgLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(testDir, ".imgpkg", "images.yml"))
	if err != nil {
		t.Fatalf("could not read images lock: %v", err)
	}

	greeting, ok := imgLock.Images[0].Annotations["greeting"]
	if !ok {
		t.Fatalf("could not find annoation greeting in images lock")
	}
	if greeting != "hello world" {
		t.Fatalf("Expected images lock to have annotation saying 'hello world', got: %s", greeting)
	}
}
