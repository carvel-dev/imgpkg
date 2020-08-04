package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultiImgpkgDirError(t *testing.T) {
	tempDir := os.TempDir()

	pushDir := filepath.Join(tempDir, "imgpkg-push-units-multi-dir")
	defer Cleanup(pushDir)

	// cleaned up via pushDir
	fooDir := filepath.Join(pushDir, "foo")
	barDir := filepath.Join(pushDir, "bar")

	// cleanup any previous state
	Cleanup(pushDir)
	err := os.Mkdir(pushDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = os.MkdirAll(filepath.Join(fooDir, ".imgpkg"), 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = os.MkdirAll(filepath.Join(barDir, ".imgpkg"), 0700)
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

	// cleanup any previous state
	Cleanup(pushDir)
	err := os.Mkdir(pushDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = os.MkdirAll(filepath.Join(fooDir, ".imgpkg"), 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{fooDir}}, ImageFlags: ImageFlags{Image: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Images cannot be pushed with a '.imgpkg' bundle directory") {
		t.Fatalf("Expected error to contain message about image with bundle dir, got: %s", err)
	}
}

func TestNestedImgpkgDirError(t *testing.T) {
	tempDir := os.TempDir()
	pushDir := filepath.Join(tempDir, "imgpkg-push-units-nested-dir")
	defer Cleanup(pushDir)

	// cleanup any previous state
	Cleanup(pushDir)
	err := os.Mkdir(pushDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	// cleaned up via push dir
	err = os.MkdirAll(filepath.Join(pushDir, "foo", ".imgpkg"), 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{pushDir}}, BundleFlags: BundleFlags{Bundle: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected '.imgpkg' dir to be a direct child") {
		t.Fatalf("Expected error to contain message about .imgpkg being a direct child, got: %s", err)
	}
}

func TestDuplicateFilepathError(t *testing.T) {
	tempDir := os.TempDir()

	pushDir := filepath.Join(tempDir, "imgpkg-push-units-dup-filepath")
	defer Cleanup(pushDir)

	// cleaned up via pushDir
	fooDir := filepath.Join(pushDir, "foo")

	// cleanup any previous state
	Cleanup(pushDir)

	err := os.MkdirAll(fooDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	someFile := filepath.Join(fooDir, "some-file.yml")
	err = ioutil.WriteFile(someFile, []byte("foo: bar"), 0600)
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
	push := PushOptions{ImageFlags: ImageFlags{"image@123456"}, OutputFlags: OutputFlags{LockFilePath: "lock-file"}}
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
