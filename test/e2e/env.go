// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"strings"
	"testing"
)

type Env struct {
	Image      string
	ImgpkgPath string
}

func BuildEnv(t *testing.T) Env {
	imgpkgPath := os.Getenv("IMGPKG_BINARY")
	if imgpkgPath == "" {
		imgpkgPath = "imgpkg"
	}

	env := Env{
		Image:      os.Getenv("IMGPKG_E2E_IMAGE"),
		ImgpkgPath: imgpkgPath,
	}
	env.Validate(t)
	return env
}

func (e Env) Validate(t *testing.T) {
	errStrs := []string{}

	if len(e.Image) == 0 {
		errStrs = append(errStrs, "Expected Image to be non-empty")
	}

	if len(errStrs) > 0 {
		t.Fatalf("%s", strings.Join(errStrs, "\n"))
	}
}
