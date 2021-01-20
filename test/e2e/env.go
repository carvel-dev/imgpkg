// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"strings"
	"testing"
)

type Env struct {
	Image          string
	ImgpkgPath     string
	RelocationRepo string
	BundleFactory  bundleFactory
	Assets         *assets
	Assert         assertion
	ImageFactory   imageFactory
	cleanupFuncs   []func()
}

func BuildEnv(t *testing.T) *Env {
	t.Helper()
	imgpkgPath := os.Getenv("IMGPKG_BINARY")
	if imgpkgPath == "" {
		imgpkgPath = "imgpkg"
	}

	assets := &assets{t: t}
	env := Env{
		Image:          os.Getenv("IMGPKG_E2E_IMAGE"),
		RelocationRepo: os.Getenv("IMGPKG_E2E_RELOCATION_REPO"),
		ImgpkgPath:     imgpkgPath,
		BundleFactory:  newBundleDir(t, assets),
		Assets:         assets,
		Assert:         assertion{t: t},
		ImageFactory: imageFactory{
			assets: assets,
			t:      t,
		},
	}
	env.Validate(t)
	return &env
}
func (e *Env) AddCleanup(f func()) {
	e.cleanupFuncs = append(e.cleanupFuncs, f)
}
func (e *Env) Cleanup() {
	e.Assets.CleanCreatedFolders()
	for i := len(e.cleanupFuncs) - 1; i >= 0; i-- {
		e.cleanupFuncs[i]()
	}
}

func (e Env) Validate(t *testing.T) {
	t.Helper()
	var errStrs []string

	if len(e.Image) == 0 {
		errStrs = append(errStrs, "Expected environment variable 'IMGPKG_E2E_IMAGE' to be non-empty. For example `export IMGPKG_E2E_IMAGE=index.docker.io/k8slt/imgpkg-test`")
	} else {
		parts := strings.SplitN(e.Image, "/", 2)
		if !(len(parts) == 2 && (strings.ContainsRune(parts[0], '.') || strings.ContainsRune(parts[0], ':'))) {
			errStrs = append(errStrs, "The IMGPKG_E2E_IMAGE environment variable did not contain a valid domain. For example `export IMGPKG_E2E_IMAGE=index.docker.io/k8slt/imgpkg-test`")
		}
	}

	if len(e.RelocationRepo) == 0 {
		errStrs = append(errStrs, "Expected environment variable 'IMGPKG_E2E_RELOCATION_REPO' to be non-empty. For example `export IMGPKG_E2E_RELOCATION_REPO=index.docker.io/k8slt/imgpkg-test-relocation`")
	} else {
		parts := strings.SplitN(e.RelocationRepo, "/", 2)
		if !(len(parts) == 2 && (strings.ContainsRune(parts[0], '.') || strings.ContainsRune(parts[0], ':'))) {
			errStrs = append(errStrs, "The IMGPKG_E2E_RELOCATION_REPO environment variable did not contain a valid domain. For example `export IMGPKG_E2E_RELOCATION_REPO=index.docker.io/k8slt/imgpkg-test-relocation`")
		}
	}

	if len(errStrs) > 0 {
		t.Fatalf("%s", strings.Join(errStrs, "\n"))
	}
}
