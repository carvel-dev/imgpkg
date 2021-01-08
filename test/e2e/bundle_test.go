// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundlePushPullAnnotation(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Assets.cleanCreatedFolders()

	bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imagesYAML)

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir})

	if !env.BundleFactory.isImageABundle(env.Image) {
		t.Fatalf("Expected to find bundle but didn't")
	}

	outDir := env.Assets.createTempFolder("bundle-annotation")

	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", outDir})

	env.Assets.validateFilesAreEqual(bundleDir, outDir, env.Assets.filesInFolder())
}

func TestPushWithFileExclusion(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Assets.cleanCreatedFolders()

	bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imagesYAML)

	env.BundleFactory.addFileToBundle("excluded-file.txt", "I will not be present in the bundle")
	env.BundleFactory.addFileToBundle(
		filepath.Join("nested-dir", "excluded-file.txt"),
		"this file will not be excluded because it is nested",
	)

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir, "--file-exclusion", "excluded-file.txt"})

	if !env.BundleFactory.isImageABundle(env.Image) {
		t.Fatalf("Expected to find bundle but didn't")
	}

	outDir := env.Assets.createTempFolder("bundle-exclusion")

	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", outDir})

	expectedFiles := []string{
		"nested-dir/excluded-file.txt",
	}
	expectedFiles = append(expectedFiles, env.Assets.filesInFolder()...)
	env.Assets.validateFilesAreEqual(bundleDir, outDir, expectedFiles)
}

func TestBundleLockFile(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Assets.cleanCreatedFolders()

	bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imagesYAML)

	bundleLockFilepath := filepath.Join(env.Assets.createTempFolder("bundle-lock"), "imgpkg-bundle-lock-test.yml")

	// push the bundle in the assets dir
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir, "--lock-output", bundleLockFilepath})

	env.Assert.isBundleLockFile(bundleLockFilepath, env.Image)

	outputDir := env.Assets.createTempFolder("bundle-pull")
	imgpkg.Run([]string{"pull", "--lock", bundleLockFilepath, "-o", outputDir})

	env.Assets.validateFilesAreEqual(bundleDir, outputDir, env.Assets.filesInFolder())
}

func TestImagePullOnBundleError(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Assets.cleanCreatedFolders()

	bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imagesYAML)

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir})

	var stderrBs bytes.Buffer

	path := env.Assets.createTempFolder("not-used")
	_, err := imgpkg.RunWithOpts([]string{"pull", "-i", env.Image, "-o", path},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	if err == nil {
		t.Fatalf("Expected incorrect flag error")
	}
	if !strings.Contains(errOut, "Expected bundle flag when pulling a bundle (hint: Use -b instead of -i for bundles)") {
		t.Fatalf("Expected error to contain message about using the wrong pull flag, got: %s", errOut)
	}
}

func TestBundlePullOnImageError(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Assets.cleanCreatedFolders()

	imageDir := env.Assets.createAndCopy("image-folder")

	imgpkg.Run([]string{"push", "-i", env.Image, "-f", imageDir})

	path := env.Assets.createTempFolder("not-used")
	var stderrBs bytes.Buffer
	_, err := imgpkg.RunWithOpts([]string{"pull", "-b", env.Image, "-o", path},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})

	if err == nil {
		t.Fatal("Expected incorrect flag error")
	}

	errOut := stderrBs.String()

	if !strings.Contains(errOut, "Expected bundle image but found plain image (hint: Did you use -i instead of -b?)") {
		t.Fatalf("Expected error to contain message about using the wrong pull flag, got: %s", errOut)
	}
}
