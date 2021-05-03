// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"testing"

	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthErr(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}

	outputDir := env.Assets.CreateTempFolder("pull-image")
	defer env.Assets.CleanCreatedFolders()

	var stderrBs bytes.Buffer

	_, err := imgpkg.RunWithOpts([]string{
		"pull", "-i", "index.docker.io/k8slt/imgpkg-test", "-o", outputDir,
		"--registry-username", "incorrect-user",
		"--registry-password", "incorrect-password",
	}, helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})

	errOut := stderrBs.String()
	require.Error(t, err)
	assert.Contains(t, errOut, "incorrect username or password")
}
