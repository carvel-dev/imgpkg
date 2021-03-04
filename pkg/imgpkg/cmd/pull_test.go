// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"

	"github.com/cppforlife/go-cli-ui/ui"
)

func TestNoImageOrBundleOrLockError(t *testing.T) {
	pull := PullOptions{OutputPath: "/tmp/some/place"}
	err := pull.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected either image or bundle reference") {
		t.Fatalf("Expected error to contain message about invalid flags, got: %s", err)
	}
}

func TestImageAndBundleAndLockError(t *testing.T) {
	pull := PullOptions{OutputPath: "/tmp/some/place", ImageFlags: ImageFlags{"image@123456"}, BundleFlags: BundleFlags{"my-bundle", false}, LockInputFlags: LockInputFlags{LockFilePath: "lockpath"}}
	err := pull.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected only one of image, bundle, or lock") {
		t.Fatalf("Expected error to contain message about invalid flags, got: %s", err)
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
