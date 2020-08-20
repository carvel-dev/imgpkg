// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	env := BuildEnv(t)
	out := Imgpkg{t, Logger{}, env.ImgpkgPath}.Run([]string{"version"})

	if !strings.Contains(out, "imgpkg version") {
		t.Fatalf("Expected to find client version")
	}
}
