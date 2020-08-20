// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"regexp"
	"testing"

	uitest "github.com/cppforlife/go-cli-ui/ui/test"
)

func TestTagList(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	assetsPath := "assets/simple-app"

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag1", "-f", assetsPath})
	tag1Digest := extractDigest(out, t)

	out = imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag2", "-f", assetsPath})
	tag2Digest := extractDigest(out, t)

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

func extractDigest(out string, t *testing.T) string {
	match := regexp.MustCompile("@(sha256:[0123456789abcdef]{64})").FindStringSubmatch(out)
	if len(match) != 2 {
		t.Fatalf("Expected to find digest in output '%s'", out)
	}
	return match[1]
}
