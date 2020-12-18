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

	"github.com/cppforlife/go-cli-ui/ui"
)

func TestNoImageOrBundleOrLockError(t *testing.T) {
	pull := PullOptions{}
	err := pull.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected either image, bundle, or lock") {
		t.Fatalf("Expected error to contain message about invalid flags, got: %s", err)
	}
}

func TestImageAndBundleAndLockError(t *testing.T) {
	pull := PullOptions{ImageFlags: ImageFlags{"image@123456"}, BundleFlags: BundleFlags{"my-bundle"}, LockInputFlags: LockInputFlags{LockFilePath: "lockpath"}}
	err := pull.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected only one of image, bundle, or lock") {
		t.Fatalf("Expected error to contain message about invalid flags, got: %s", err)
	}
}

func TestInvalidBundleLockKind(t *testing.T) {
	tempDir := os.TempDir()

	workingDir := filepath.Join(tempDir, "imgpkg-pull-units-invalid-kind")
	defer Cleanup(workingDir)
	err := os.MkdirAll(workingDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	lockFilePath := filepath.Join(workingDir, "bundlelock.yml")
	ioutil.WriteFile(lockFilePath, []byte(`
---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: invalid-value
spec:
  image:
    url: index.docker.io/k8slt/test
    tag: latest
`), 0600)

	pull := PullOptions{LockInputFlags: LockInputFlags{LockFilePath: lockFilePath}}
	err = pull.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	reg := regexp.MustCompile("Invalid `kind` in lockfile at .*imgpkg-pull-units-invalid-kind/bundlelock\\.yml. Expected: BundleLock, got: invalid-value")
	if !reg.MatchString(err.Error()) {
		t.Fatalf("Expected error to contain message about invalid bundlelock kind, got: %s", err)
	}
}

func Test_Invalid_Args_Passed(t *testing.T) {
	confUI := ui.NewConfUI(ui.NewNoopLogger())
	defer confUI.Flush()

	imgpkgCmd := NewDefaultImgpkgCmd(confUI)
	imgpkgCmd.SetArgs([]string{"pull", "k8slt/image", "-o", "/tmp"})
	err := imgpkgCmd.Execute()
	if err == nil {
		t.Fatalf("Expected error from executing imgpkg pull command: %v", err)
	}

	expected := "command 'imgpkg pull' does not accept extra arguments 'k8slt/image'"
	if expected != err.Error() {
		t.Fatalf("\nExpceted: %s\nGot: %s", expected, err.Error())
	}
}
