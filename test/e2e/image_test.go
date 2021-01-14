// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"path/filepath"
	"testing"
)

func TestPushPull(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	testDir := env.Assets.CreateTempFolder("imgpkg-test-basic")

	imgpkg.Run([]string{"push", "-i", env.Image, "-f", env.Assets.SimpleAppDir()})
	imgpkg.Run([]string{"pull", "-i", env.Image, "-o", testDir})

	env.Assets.ValidateFilesAreEqual(env.Assets.SimpleAppDir(), testDir, []string{
		"README.md",
		"LICENSE",
		"config/config.yml",
		"config/inner-dir/README.txt",
	})
}

func TestPushMultipleFiles(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	imgpkg.Run([]string{
		"push", "-i", env.Image,
		"-f", filepath.Join(env.Assets.SimpleAppDir(), "LICENSE"),
		"-f", filepath.Join(env.Assets.SimpleAppDir(), "README.md"),
		"-f", filepath.Join(env.Assets.SimpleAppDir(), "config"),
	})

	testDir := env.Assets.CreateTempFolder("imgpkg-test-multiple-files")
	imgpkg.Run([]string{"pull", "-i", env.Image, "-o", testDir})

	expectedFiles := map[string]string{
		"README.md":                   "README.md",
		"LICENSE":                     "LICENSE",
		"config/config.yml":           "config.yml",
		"config/inner-dir/README.txt": "inner-dir/README.txt",
	}

	for assetFile, downloadedFile := range expectedFiles {
		compareFiles(t, filepath.Join(env.Assets.SimpleAppDir(), assetFile), filepath.Join(testDir, downloadedFile))
	}
}
