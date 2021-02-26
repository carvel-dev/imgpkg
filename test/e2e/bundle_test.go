// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/test/helpers"
)

func TestBundlePushPullAnnotation(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir})

	ref, _ := name.NewTag(env.Image, name.WeakValidation)
	image, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		t.Fatalf("Error getting remote image: %s", err)
	}

	config, err := image.ConfigFile()
	if err != nil {
		t.Fatalf("Error getting manifest: %s", err)
	}

	if _, found := config.Config.Labels["dev.carvel.imgpkg.bundle"]; !found {
		t.Fatalf("Expected to find bundle but didn't")
	}

	outDir := env.Assets.CreateTempFolder("bundle-annotation")
	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", outDir})

	env.Assets.ValidateFilesAreEqual(bundleDir, outDir, env.Assets.FilesInFolder())
}

func TestPushWithFileExclusion(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)

	env.BundleFactory.AddFileToBundle("excluded-file.txt", "I will not be present in the bundle")
	env.BundleFactory.AddFileToBundle(
		filepath.Join("nested-dir", "excluded-file.txt"),
		"this file will not be excluded because it is nested",
	)

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir, "--file-exclusion", "excluded-file.txt"})

	outDir := env.Assets.CreateTempFolder("bundle-exclusion")
	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", outDir})

	expectedFiles := []string{
		"nested-dir/excluded-file.txt",
	}
	expectedFiles = append(expectedFiles, env.Assets.FilesInFolder()...)
	env.Assets.ValidateFilesAreEqual(bundleDir, outDir, expectedFiles)
}

func TestBundleLockFile(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)

	bundleLockFilepath := filepath.Join(env.Assets.CreateTempFolder("bundle-lock"), "imgpkg-bundle-lock-test.yml")

	// push the bundle in the assets dir
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir, "--lock-output", bundleLockFilepath})

	bundleBs, err := ioutil.ReadFile(bundleLockFilepath)
	if err != nil {
		t.Fatalf("Could not read bundle lock file: %s", err)
	}

	// Keys are written in alphabetical order
	expectedYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
bundle:
  image: %s@sha256:[A-Fa-f0-9]{64}
  tag: latest
kind: BundleLock
`, env.Image)

	if !regexp.MustCompile(expectedYml).Match(bundleBs) {
		t.Fatalf("Regex did not match; diff expected...actual:\n%v\n", helpers.DiffText(expectedYml, string(bundleBs)))
	}

	outputDir := env.Assets.CreateTempFolder("bundle-pull")
	imgpkg.Run([]string{"pull", "--lock", bundleLockFilepath, "-o", outputDir})

	env.Assets.ValidateFilesAreEqual(bundleDir, outputDir, env.Assets.FilesInFolder())
}

func TestImagePullOnBundleError(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir})

	var stderrBs bytes.Buffer

	path := env.Assets.CreateTempFolder("not-used")
	_, err := imgpkg.RunWithOpts([]string{"pull", "-i", env.Image, "-o", path},
		helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	if err == nil {
		t.Fatalf("Expected incorrect flag error")
	}
	if !strings.Contains(errOut, "Expected bundle flag when pulling a bundle (hint: Use -b instead of -i for bundles)") {
		t.Fatalf("Expected error to contain message about using the wrong pull flag, got: %s", errOut)
	}
}

func TestBundlePullOnImageError(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	imageDir := env.Assets.CreateAndCopySimpleApp("image-folder")
	imgpkg.Run([]string{"push", "-i", env.Image, "-f", imageDir})

	path := env.Assets.CreateTempFolder("not-used")
	var stderrBs bytes.Buffer
	_, err := imgpkg.RunWithOpts([]string{"pull", "-b", env.Image, "-o", path},
		helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})

	if err == nil {
		t.Fatal("Expected incorrect flag error")
	}

	errOut := stderrBs.String()

	if !strings.Contains(errOut, "Expected bundle image but found plain image (hint: Did you use -i instead of -b?)") {
		t.Fatalf("Expected error to contain message about using the wrong pull flag, got: %s", errOut)
	}
}
