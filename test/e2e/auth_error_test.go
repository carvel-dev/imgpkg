// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"strings"
	"testing"

	"github.com/k14s/imgpkg/test/helpers"
)

func TestAuthErr(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}

	outputDir := env.Assets.CreateTempFolder("pull-image")
	defer env.Assets.CleanCreatedFolders()

	var stderrBs bytes.Buffer

	_, err := imgpkg.RunWithOpts([]string{
		"pull", "-i", "index.docker.io/k8slt/imgpkg-test", "-o", outputDir,
		"--registry-username", "incorrect-user",
		"--registry-password", "incorrect-password",
	}, helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})

	errOut := stderrBs.String()

	if err == nil {
		t.Fatalf("Expected auth error")
	}
	if !strings.Contains(errOut, "incorrect username or password") {
		t.Fatalf("Expected auth error explanation in output '%s'", errOut)
	}
}
