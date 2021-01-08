// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
)

func TestDeterministicPush(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	assetsPath := "assets/simple-app"

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag1", "-f", assetsPath})
	tag1Digest := extractDigest(t, out)

	out = imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag2", "-f", assetsPath})
	tag2Digest := extractDigest(t, out)

	if tag1Digest != tag2Digest {
		t.Fatalf("Digests do not match, hence non-deterministic")
	}
}
