// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"

	"github.com/k14s/imgpkg/test/helpers"
)

func TestVersion(t *testing.T) {
	env := helpers.BuildEnv(t)
	out := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}.Run([]string{"version"})

	if !strings.Contains(out, "imgpkg version") {
		t.Fatalf("Expected to find client version")
	}
}
