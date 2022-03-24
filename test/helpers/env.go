// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type Env struct {
	Image          string
	ImgpkgPath     string
	RelocationRepo string
	BundleFactory  BundleFactory
	Assets         *Assets
	Assert         Assertion
	ImageFactory   ImageFactory
	Logger         *Logger
	cleanupFuncs   []func()
}

func BuildEnv(t *testing.T) *Env {
	t.Helper()
	imgpkgPath := os.Getenv("IMGPKG_BINARY")
	if imgpkgPath == "" {
		imgpkgPath = "imgpkg"
	}

	assets := &Assets{T: t}
	logger := &Logger{LogLevel: LogDebug}
	env := Env{
		Image:          os.Getenv("IMGPKG_E2E_IMAGE"),
		RelocationRepo: os.Getenv("IMGPKG_E2E_RELOCATION_REPO"),
		ImgpkgPath:     imgpkgPath,
		BundleFactory:  NewBundleDir(t, assets),
		Assets:         assets,
		Assert: Assertion{
			T:                    t,
			logger:               logger,
			signatureKeyLocation: filepath.Join(filepath.Dir(imgpkgPath), "tmp"),
		},
		Logger: logger,
		ImageFactory: ImageFactory{
			Assets:               assets,
			T:                    t,
			signatureKeyLocation: filepath.Join(filepath.Dir(imgpkgPath), "tmp"),
			logger:               logger,
		},
	}
	env.Validate(t)
	return &env
}

func (e *Env) UpdateT(t *testing.T) {
	e.BundleFactory.t = t
	e.Assert.T = t
	e.Assets.T = t
	e.ImageFactory.T = t
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

	require.Len(t, errStrs, 0)
}
