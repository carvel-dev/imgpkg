package cmd

import (
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

func Cleanup(dirs ...string) {
	for _, dir := range dirs {
		os.RemoveAll(dir)
	}
}
