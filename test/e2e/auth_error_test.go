// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

func TestAuthErr(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}

	outputDir := env.Assets.CreateTempFolder("pull-image")
	defer env.Assets.CleanCreatedFolders()

	var stderrBs bytes.Buffer

	registry := helpers.NewFakeRegistry(t, env.Logger)
	registry.WithBasicAuth("some-user", "some-password")
	defer registry.CleanUp()

	_, err := imgpkg.RunWithOpts([]string{
		"pull", "-i", registry.ReferenceOnTestServer("imgpkg-test"), "-o", outputDir,
		"--registry-username", "incorrect-user",
		"--registry-password", "incorrect-password",
	}, helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})

	errOut := stderrBs.String()
	require.Error(t, err)
	assert.Contains(t, errOut, "incorrect username or password")
}
