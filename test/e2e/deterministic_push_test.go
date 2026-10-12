// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"runtime"
	"testing"

	"carvel.dev/imgpkg/test/helpers"
	"github.com/stretchr/testify/require"
)

func TestDeterministicPush(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Image pushed on windows results in a different sha due to backslashes used on filesystem. Skipping for now until fixed.")
	}

	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	assetsPath := "assets/simple-app"

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag1", "-f", assetsPath})
	tag1Digest := helpers.ExtractDigest(t, out)

	// This digest includes gzip output generated with Go 1.27 and should match
	// across non-Windows environments using the same Go version.
	require.Equal(t, "sha256:7dfc87f48adfa9f12474f84cdf30efdfc750acb4c1249d2321ad5bf479b08857", tag1Digest, "Digest should match in all environments")

	out = imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag2", "-f", assetsPath})
	tag2Digest := helpers.ExtractDigest(t, out)

	require.Equal(t, tag1Digest, tag2Digest, "Digests do not match, hence non-deterministic")
}
