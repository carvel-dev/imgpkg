// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

func TestTagResolve(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag1", "-f", env.Assets.SimpleAppDir()})
	tag1Digest := helpers.ExtractDigest(t, out)

	resolvedDigestOut := imgpkg.Run([]string{"tag", "resolve", "-i", env.Image + ":tag1"})
	require.Contains(t, resolvedDigestOut, "@"+tag1Digest)

	// When ref is in digest format, returns same thing
	resolvedDigestOut2 := imgpkg.Run([]string{"tag", "resolve", "-i", resolvedDigestOut})
	require.Equal(t, resolvedDigestOut2, resolvedDigestOut)

	// With wrong digest, registry is expected to be contacted to check
	_, err := imgpkg.RunWithOpts([]string{"tag", "resolve", "-i",
		env.Image + "@sha256:8f335768880da6baf72b70c701002b45f4932acae8d574dedfddaf967ac3ac90"},
		helpers.RunOpts{AllowError: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "8f335768880da6baf72b70c701002b45f4932acae8d574dedfddaf967ac3ac90")
}
