// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	uitest "github.com/cppforlife/go-cli-ui/ui/test"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

func TestTagList(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag1", "-f", env.Assets.SimpleAppDir()})
	tag1Digest := helpers.ExtractDigest(t, out)

	out = imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag2", "-f", env.Assets.SimpleAppDir()})
	tag2Digest := helpers.ExtractDigest(t, out)

	expectedTags := map[string]string{"tag1": tag1Digest, "tag2": tag2Digest}

	{ // Without digests
		out := imgpkg.Run([]string{"tag", "list", "-i", env.Image, "--json"})
		resp := uitest.JSONUIFromBytes(t, []byte(out))

		for name := range expectedTags {
			var found bool
			for _, row := range resp.Tables[0].Rows {
				if row["name"] == name {
					found = true
					require.Equal(t, "", row["digest"])
					break
				}
			}
			require.Truef(t, found, "Expected to find tag '%s'", name)
		}
	}

	{ // With digests
		out := imgpkg.Run([]string{"tag", "list", "-i", env.Image, "--digests=true", "--json"})
		resp := uitest.JSONUIFromBytes(t, []byte(out))

		for name, digest := range expectedTags {
			var found bool
			for _, row := range resp.Tables[0].Rows {
				if row["name"] == name {
					found = true
					require.Equal(t, digest, row["digest"])
					break
				}
			}
			require.Truef(t, found, "Expected to find tag '%s'", name)
		}
	}
}
