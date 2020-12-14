// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	lf "github.com/k14s/imgpkg/pkg/imgpkg/lockfiles"
)

const emptyImagesYaml = `apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images: []`

func TestMultiImgpkgDirError(t *testing.T) {
	tempDir := os.TempDir()

	pushDir := filepath.Join(tempDir, "imgpkg-push-units-multi-dir")
	defer Cleanup(pushDir)

	// cleaned up via pushDir
	fooDir := filepath.Join(pushDir, "foo")
	barDir := filepath.Join(pushDir, "bar")

	// cleanup any previous state
	Cleanup(pushDir)
	err := os.MkdirAll(fooDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = os.MkdirAll(barDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = createBundleDir(fooDir, "")
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = createBundleDir(barDir, "")
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{fooDir, barDir}}, BundleFlags: BundleFlags{Bundle: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected one '.imgpkg' dir, got 2") {
		t.Fatalf("Expected error to contain message about multiple .imgpkg dirs, got: %s", err)
	}
}

func TestInvalidLockKindError(t *testing.T) {
	tempDir := os.TempDir()

	pushDir := filepath.Join(tempDir, "imgpkg-push-units-invalid-kind")
	defer Cleanup(pushDir)

	// cleaned up via pushDir
	bundleDir := filepath.Join(pushDir, "bundle-dir")
	err := os.MkdirAll(bundleDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	// This images.yml file came from kbld resolving nginx
	err = createBundleDir(bundleDir, `apiVersion: kbld.k14s.io/v1alpha1
kind: Config
minimumRequiredVersion: 0.27.0
overrides:
  - image: nginx
newImage: index.docker.io/library/nginx@sha256:6b1daa9462046581ac15be20277a7c75476283f969cb3a61c8725ec38d3b01c3
preresolved: true
`)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{bundleDir}}, BundleFlags: BundleFlags{Bundle: "org/repo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	reg := regexp.MustCompile("Invalid `kind` in lockfile at .*bundle-dir/\\.imgpkg/images\\.yml. Expected: ImagesLock, got: Config")
	if !reg.MatchString(err.Error()) {
		t.Fatalf("Expected error to contain message about invalid images.yml kind, got: %s", err)
	}
}

func TestImageWithImgpkgDirError(t *testing.T) {
	tempDir := os.TempDir()

	pushDir := filepath.Join(tempDir, "imgpkg-push-units-image-dir")
	defer Cleanup(pushDir)

	// cleaned up via pushDir
	fooDir := filepath.Join(pushDir, "foo")
	err := os.MkdirAll(fooDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = createBundleDir(fooDir, "")
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{fooDir}}, ImageFlags: ImageFlags{Image: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	reg := regexp.MustCompile("Images cannot be pushed with '.imgpkg' directories.*, consider using a bundle")
	if !reg.MatchString(err.Error()) {
		t.Fatalf("Expected error to contain message about image with bundle dir, got: %s", err)
	}
}

func TestNestedImgpkgDirError(t *testing.T) {
	tempDir := os.TempDir()
	pushDir := filepath.Join(tempDir, "imgpkg-push-units-nested-dir")
	defer Cleanup(pushDir)

	// cleaned up via push dir
	fooDir := filepath.Join(pushDir, "foo")
	err := os.MkdirAll(fooDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = createBundleDir(fooDir, "")
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{pushDir}}, BundleFlags: BundleFlags{Bundle: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected '.imgpkg' directory, to be a direct child of one of") {
		t.Fatalf("Expected error to contain message about .imgpkg being a direct child, got: %s", err)
	}
}

func TestBundleFlagWithoutDirectoryError(t *testing.T) {
	tempDir := os.TempDir()
	pushDir := filepath.Join(tempDir, "imgpkg-push-units-bundle-without-dir")
	defer Cleanup(pushDir)

	// cleanup any previous state
	Cleanup(pushDir)
	err := os.Mkdir(pushDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{pushDir}}, BundleFlags: BundleFlags{Bundle: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected one '.imgpkg' dir, got 0") {
		t.Fatalf("Expected error to contain message about missing .imgpkg dir, got: %s", err)
	}
}

func TestDuplicateFilepathError(t *testing.T) {
	tempDir := os.TempDir()

	pushDir := filepath.Join(tempDir, "imgpkg-push-units-dup-filepath")
	defer Cleanup(pushDir)

	// cleaned up via pushDir
	fooDir := filepath.Join(pushDir, "foo")
	err := os.MkdirAll(fooDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	someFile := filepath.Join(fooDir, "some-file.yml")
	err = ioutil.WriteFile(someFile, []byte("foo: bar"), 0600)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = createBundleDir(fooDir, "")
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	// duplicate someFile.yaml by including it directly and with the dir fooDir
	push := PushOptions{FileFlags: FileFlags{Files: []string{someFile, fooDir}}, BundleFlags: BundleFlags{Bundle: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Found duplicate paths:") {
		t.Fatalf("Expected error to contain message about a duplicate filepath, got: %s", err)
	}
}

func TestNoImageOrBundleError(t *testing.T) {
	push := PushOptions{}
	err := push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected either image or bundle") {
		t.Fatalf("Expected error to contain message about invalid flags, got: %s", err)
	}
}

func TestImageAndBundleError(t *testing.T) {
	push := PushOptions{ImageFlags: ImageFlags{"image@123456"}, BundleFlags: BundleFlags{"my-bundle"}}
	err := push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected only one of image or bundle") {
		t.Fatalf("Expected error to contain message about invalid flags, got: %s", err)
	}
}

func TestImageAndBundleLockError(t *testing.T) {
	push := PushOptions{ImageFlags: ImageFlags{"image@123456"}, LockOutputFlags: LockOutputFlags{LockFilePath: "lock-file"}}
	err := push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Lock output is not compatible with image, use bundle for lock output") {
		t.Fatalf("Expected error to contain message about invalid flags, got: %s", err)
	}
}

func Cleanup(dirs ...string) {
	for _, dir := range dirs {
		os.RemoveAll(dir)
	}
}

func TestUnresolvedImageRefError(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "imgpkg-unresolved-ref-test")
	defer Cleanup(testDir)

	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	imagesYml := `apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: index.docker.io/library/nginx@sha256:36b74457bccb56fbf8b05f79c85569501b721d4db813b684391d63e02287c0b2
  - image: docker.io/another-app:v1.0`

	err = createBundleDir(testDir, imagesYml)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{testDir}}, BundleFlags: BundleFlags{Bundle: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected ref to be in digest form, got") {
		t.Fatalf("Expected error to contain message about an image reference not being in digest form, got: %s", err)
	}
}

func createBundleDir(loc, imagesYaml string) error {
	if imagesYaml == "" {
		imagesYaml = emptyImagesYaml
	}

	bundleDir := filepath.Join(loc, lf.BundleDir)
	err := os.Mkdir(bundleDir, 0700)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(bundleDir, lf.ImageLockFile), []byte(imagesYaml), 0600)
}
