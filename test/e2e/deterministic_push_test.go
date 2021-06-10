// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/require"
)

func TestDeterministicPush(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	assetsPath := "assets/simple-app"

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag1", "-f", assetsPath})
	tag1Digest := helpers.ExtractDigest(t, out)

	// This expected digest should be the same regardless which OS imgpkg runs on
	require.Equal(t, tag1Digest, "sha256:ceef30cbdce418efde0284f446df9cec9e535adcd6e1010dad30ddae1dc9367b", "Digest should match in all environments")

	out = imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag2", "-f", assetsPath})
	tag2Digest := helpers.ExtractDigest(t, out)

	require.Equal(t, tag1Digest, tag2Digest, "Digests do not match, hence non-deterministic")
}
