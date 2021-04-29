// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	uitest "github.com/cppforlife/go-cli-ui/ui/test"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/require"
)

func TestTagList(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag1", "-f", env.Assets.SimpleAppDir()})
	tag1Digest := helpers.ExtractDigest(t, out)

	out = imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag2", "-f", env.Assets.SimpleAppDir()})
	tag2Digest := helpers.ExtractDigest(t, out)

	out = imgpkg.Run([]string{"tag", "list", "-i", env.Image, "--json"})
	resp := uitest.JSONUIFromBytes(t, []byte(out))

	expectedTags := map[string]string{"tag1": tag1Digest, "tag2": tag2Digest}

	for name, digest := range expectedTags {
		var found bool
		for _, row := range resp.Tables[0].Rows {
			if row["name"] == name {
				found = true
				require.Equal(t, row["digest"], digest)
				break
			}
		}
		require.Truef(t, found, "Expected to find tag '%s'", name)
	}
}
