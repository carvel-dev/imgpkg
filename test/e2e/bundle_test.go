// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func TestBundlePushPullAnnotation(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	assetsDir := filepath.Join("assets", "simple-app")
	bundleDir, err := createBundleDir(assetsDir, bundleYAML, imagesYAML)
	defer os.RemoveAll(bundleDir)
	if err != nil {
		t.Fatalf("Creating bundle directory: %s", err.Error())
	}

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsDir})

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
		t.Fatalf("Expected config to contain bundle label, instead had: %v", config.Config.Labels)
	}

	outDir := filepath.Join(os.TempDir(), "bundle-pull")
	if err := os.Mkdir(outDir, 0600); err != nil {
		t.Fatalf("Error creating temp dir: %s", err)
	}
	defer os.RemoveAll(outDir)

	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", outDir})

	expectedFiles := []string{
		".imgpkg/bundle.yml",
		".imgpkg/images.yml",
		"README.md",
		"LICENSE",
		"config/config.yml",
		"config/inner-dir/README.txt",
	}

	for _, file := range expectedFiles {
		compareFiles(filepath.Join(assetsDir, file), filepath.Join(outDir, file), t)
	}
}

func TestBundleLockFile(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	assetsDir := filepath.Join("assets", "simple-app")
	bundleDir, err := createBundleDir(assetsDir, bundleYAML, imagesYAML)
	defer os.RemoveAll(bundleDir)
	if err != nil {
		t.Fatalf("Creating bundle directory: %s", err.Error())
	}

	bundleLockFilepath := filepath.Join(os.TempDir(), "imgpkg-bundle-lock-test.yml")
	defer os.RemoveAll(bundleLockFilepath)

	// push the bundle in the assets dir
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsDir, "--lock-output", bundleLockFilepath})

	bundleBs, err := ioutil.ReadFile(bundleLockFilepath)
	if err != nil {
		t.Fatalf("Could not read bundle lock file: %s", err)
	}

	expectedYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: BundleLock
spec:
  image:
    url: (%s/)?%s@sha256:[A-Fa-f0-9]{64}
    tag: latest
`, name.DefaultRegistry, env.Image)

	if !regexp.MustCompile(expectedYml).Match(bundleBs) {
		t.Fatalf("Regex did not match; diff expected...actual:\n%v\n", diffText(expectedYml, string(bundleBs)))
	}

	outputDir := filepath.Join(os.TempDir(), "imgpkg-bundle-lock-pull")
	defer os.RemoveAll(outputDir)
	imgpkg.Run([]string{"pull", "--lock", bundleLockFilepath, "-o", outputDir})

	expectedFiles := []string{
		"README.md",
		"LICENSE",
		"config/config.yml",
		"config/inner-dir/README.txt",
		".imgpkg/bundle.yml",
		".imgpkg/images.yml",
	}

	for _, file := range expectedFiles {
		compareFiles(filepath.Join(assetsDir, file), filepath.Join(outputDir, file), t)
	}

}

func TestImagePullOnBundleError(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	assetsDir := filepath.Join("assets", "simple-app")

	bundleDir, err := createBundleDir(assetsDir, bundleYAML, imagesYAML)
	defer os.RemoveAll(bundleDir)
	if err != nil {
		t.Fatalf("Creating bundle directory: %s", err.Error())
	}

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsDir})

	var stderrBs bytes.Buffer

	path := "/tmp/imgpkg-test-pull-bundle-error"
	defer os.RemoveAll(path)
	_, err = imgpkg.RunWithOpts([]string{"pull", "-i", env.Image, "-o", path},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	if err == nil {
		t.Fatalf("Expected incorrect flag error")
	}
	if !strings.Contains(errOut, "Expected bundle flag when pulling a bundle, please use -b instead of --image") {
		t.Fatalf("Expected error to contain message about using the wrong pull flag, got: %s", errOut)
	}
}

func TestBundlePullOnImageError(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	assetsPath := filepath.Join("assets", "simple-app")
	path := filepath.Join("tmp", "imgpkg-test-pull-image-error")

	cleanUp := func() { os.RemoveAll(path) }
	cleanUp()
	defer cleanUp()

	var stderrBs bytes.Buffer

	imgpkg.Run([]string{"push", "-i", env.Image, "-f", assetsPath})
	_, err := imgpkg.RunWithOpts([]string{"pull", "-b", env.Image, "-o", path},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})

	if err == nil {
		t.Fatal("Expected incorrect flag error")
	}

	errOut := stderrBs.String()

	if !strings.Contains(errOut, "Expected image flag when pulling an image or index, please use --image instead of -b") {
		t.Fatalf("Expected error to contain message about using the wrong pull flag, got: %s", errOut)
	}
}
