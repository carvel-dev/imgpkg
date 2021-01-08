// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

func TestCopyImageToRepoDestinationAndOutputImageLockFileAndPreserverImageTag(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Assets.cleanCreatedFolders()

	// create generic image
	tag := time.Now().UnixNano()
	imageDigest := env.ImageFactory.pushSimpleAppImageWithRandomFile(imgpkg, fmt.Sprintf("%s:%d", env.Image, tag))

	lockOutputPath := filepath.Join(os.TempDir(), "image-relocate-lock.yml")
	defer os.Remove(lockOutputPath)

	// copy via create ref
	imgpkg.Run([]string{"copy", "--image", fmt.Sprintf("%s:%v", env.Image, tag),
		"--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	err := env.ImageFactory.assertImagesLock(lockOutputPath, []lockconfig.ImageRef{{Image: expectedRef}})
	if err != nil {
		t.Fatalf("validating Images Lock file: %s", err)
	}

	if err := validateImagesPresenceInRegistry([]string{env.RelocationRepo + imageDigest}); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}

	if err := validateImagesPresenceInRegistry([]string{fmt.Sprintf("%s:%v", env.RelocationRepo, tag)}); err == nil {
		t.Fatalf("expected not to find image with tag '%v', but did", tag)
	}
}

func TestCopyImageInputToTarFileAndToADifferentRepoCheckImageLockIsGenerated(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-image-tar")
	tarFilePath := filepath.Join(testDir, "image.tar")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image
	assetsPath := filepath.Join("assets", "simple-app")

	// force digest to change so test is meaningful
	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	tag := fmt.Sprintf("%d", time.Now().UnixNano())
	tagRef := fmt.Sprintf("%s:%s", env.Image, tag)
	out := imgpkg.Run([]string{"push", "--tty", "-i", tagRef, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(t, out))

	// copy to tar
	imgpkg.Run([]string{"copy", "-i", tagRef, "--to-tar", tarFilePath})

	lockOutputPath := filepath.Join(os.TempDir(), "relocate-from-tar-lock.yml")
	defer os.Remove(lockOutputPath)

	// copy from tar to repo
	imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	imgLock, err := lockconfig.NewImagesLockFromPath(lockOutputPath)
	if err != nil {
		t.Fatalf("could not read lock-output: %v", err)
	}

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	if imgLock.Images[0].Image != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'", imgLock.Images[0].Image, expectedRef)
	}

	if err := validateImageLockApiVersionAndKind(imgLock); err != nil {
		t.Fatal(err.Error())
	}

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigest}
	if err := validateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyErrorsWhenCopyImageUsingBundleFlag(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// create generic image
	assetsPath := filepath.Join("assets", "simple-app")

	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(t, out))
	imageDigestRef := env.Image + imageDigest

	var stderrBs bytes.Buffer
	_, err = imgpkg.RunWithOpts([]string{"copy", "-b", imageDigestRef, "--to-tar", "fake_path"},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	if err == nil {
		t.Fatalf("Expected incorrect flag error")
	}

	if !strings.Contains(errOut, "Expected bundle image but found plain image (hint: Did you use -i instead of -b?)") {
		t.Fatalf("Expected error to contain message about using the wrong copy flag, got: %s", errOut)
	}
}

func TestCopyErrorsWhenCopyToTarAndGenerateOutputLockFile(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	_, err := imgpkg.RunWithOpts(
		[]string{"copy", "--tty", "-i", env.Image, "--to-tar", "file", "--lock-output", "bogus"},
		RunOpts{AllowError: true},
	)

	if err == nil || !strings.Contains(err.Error(), "output lock file with tar destination") {
		t.Fatalf("expected copy to fail when --lock-output is provided with a tar destination, got %v", err)
	}
}

func addRandomFile(dir string) (string, error) {
	randFile := filepath.Join(dir, "rand.yml")
	randContents := fmt.Sprintf("%d", time.Now().UnixNano())
	err := ioutil.WriteFile(randFile, []byte(randContents), 0700)
	if err != nil {
		return "", err
	}

	return randFile, nil
}

func validateImagesPresenceInRegistry(refs []string) error {
	for _, refString := range refs {
		ref, _ := name.ParseReference(refString)
		if _, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
			return fmt.Errorf("validating image %s: %v", refString, err)
		}
	}
	return nil
}

func validateBundleLockApiVersionAndKind(bLock lockconfig.BundleLock) error {
	// Do not replace bundleLockKind or bundleLockAPIVersion with consts
	// BundleLockKind or BundleLockAPIVersion.
	// This is done to prevent updating the const.
	bundleLockKind := "BundleLock"
	bundleLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if bLock.APIVersion != bundleLockAPIVersion {
		return fmt.Errorf("expected apiVersion to equal: %s, but got: %s", bundleLockAPIVersion, bLock.APIVersion)
	}

	if bLock.Kind != bundleLockKind {
		return fmt.Errorf("expected Kind to equal: %s, but got: %s", bundleLockKind, bLock.Kind)
	}
	return nil
}

func validateImageLockApiVersionAndKind(iLock lockconfig.ImagesLock) error {
	// Do not replace imageLockKind or imagesLockAPIVersion with consts
	// ImagesLockKind or ImagesLockAPIVersion.
	// This is done to prevent updating the const.
	imagesLockKind := "ImagesLock"
	imagesLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if iLock.APIVersion != imagesLockAPIVersion {
		return fmt.Errorf("expected apiVersion to equal: %s, but got: %s", imagesLockAPIVersion, iLock.APIVersion)
	}

	if iLock.Kind != imagesLockKind {
		return fmt.Errorf("expected Kind to equal: %s, but got: %s", imagesLockKind, iLock.Kind)
	}
	return nil
}
