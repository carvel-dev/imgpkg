// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const emptyImagesYaml = `apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
`

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

	expected := "Images cannot be pushed with '.imgpkg' directories, consider using --bundle (-b) option"
	if err.Error() != expected {
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

func createBundleDir(loc, imagesYaml string) error {
	if imagesYaml == "" {
		imagesYaml = emptyImagesYaml
	}

	bundleDir := filepath.Join(loc, ".imgpkg")
	err := os.Mkdir(bundleDir, 0700)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(bundleDir, "images.yml"), []byte(imagesYaml), 0600)
}
