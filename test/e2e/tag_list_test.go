// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	uitest "github.com/cppforlife/go-cli-ui/ui/test"
)

func TestTagList(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag1", "-f", env.Assets.SimpleAppDir()})
	tag1Digest := extractDigest(t, out)

	out = imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag2", "-f", env.Assets.SimpleAppDir()})
	tag2Digest := extractDigest(t, out)

	out = imgpkg.Run([]string{"tag", "list", "-i", env.Image, "--json"})
	resp := uitest.JSONUIFromBytes(t, []byte(out))

	expectedTags := map[string]string{"tag1": tag1Digest, "tag2": tag2Digest}

	for name, digest := range expectedTags {
		var found bool
		for _, row := range resp.Tables[0].Rows {
			if row["name"] == name {
				found = true
				if row["digest"] != digest {
					t.Fatalf("Expected digest for tag '%s' does not match", name)
				}
				break
			}
		}
		if !found {
			t.Fatalf("Expected to find tag '%s'", name)
		}
	}
}
