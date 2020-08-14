// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const bundleYAML = `---
apiVersion: pkgx.k14s.io/v1alpha1
kind: Bundle
metadata:
  name: my-app
authors:
- name: blah
  email: blah@blah.com
websites:
- url: blah.com
`
const imagesYAML = `---
apiVersion: pkgx.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images: []
`

func TestBundlePushPullAnnotation(t *testing.T) {
	// Do some setup
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}}
	assetsDir := filepath.Join("assets", "simple-app")
	bundleDir, err := createBundleDir(assetsDir)
	defer os.RemoveAll(bundleDir)
	if err != nil {
		t.Fatalf("Creating bundle directory: %s", err.Error())
	}

	// push the bundle in the assets dir
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsDir})

	// Validate bundle annotation is present
	ref, _ := name.NewTag(env.Image, name.WeakValidation)
	image, err := remote.Image(ref)
	if err != nil {
		t.Fatalf("Error getting remote image in test: %s", err)
	}

	manifestBs, err := image.RawManifest()
	if err != nil {
		t.Fatalf("Error getting manifest in test: %s", err)
	}

	var manifest v1.Manifest
	err = json.Unmarshal(manifestBs, &manifest)
	if err != nil {
		t.Fatalf("Error unmarshaling manifest in test: %s", err)
	}

	if val, found := manifest.Annotations["io.k14s.imgpkg.bundle"]; !found || val != "true" {
		t.Fatalf("Expected manifest to contain bundle annotation, instead had: %v", manifest.Annotations)
	}

	outDir := filepath.Join(os.TempDir(), "bundle-pull")
	if err := os.Mkdir(outDir, 0600); err != nil {
		t.Fatalf("Error creating temp dir")
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
	imgpkg := Imgpkg{t, Logger{}}
	assetsDir := filepath.Join("assets", "simple-app")
	bundleDir, err := createBundleDir(assetsDir)
	defer os.RemoveAll(bundleDir)
	if err != nil {
		t.Fatalf("Creating bundle directory: %s", err.Error())
	}

	bundleLock := filepath.Join(os.TempDir(), "imgpkg-bundle-lock-test.yml")
	defer os.RemoveAll(bundleLock)

	// push the bundle in the assets dir
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsDir, "--lock-output", bundleLock})

	bundleBs, err := ioutil.ReadFile(bundleLock)
	if err != nil {
		t.Fatalf("Could not read bundle lock file in test: %s", err)
	}

	expectedYml := fmt.Sprintf(`---
apiVersion: imgpkg.k14s.io/v1alpha1
kind: BundleLock
spec:
  image:
    url: %s@sha256:[A-Fa-f0-9]{64}
    tag: latest
`, env.Image)

	if !regexp.MustCompile(expectedYml).Match(bundleBs) {
		t.Fatalf("Expected BundleLock to match:\n\n %s\n\n, got:\n\n %s\n", expectedYml, string(bundleBs))
	}

	outputDir := filepath.Join(os.TempDir(), "imgpkg-bundle-lock-pull")
	defer os.RemoveAll(outputDir)
	imgpkg.Run([]string{"pull", "--lock", bundleLock, "-o", outputDir})

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
	imgpkg := Imgpkg{t, Logger{}}
	assetsDir := filepath.Join("assets", "simple-app")

	bundleDir, err := createBundleDir(assetsDir)
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
	imgpkg := Imgpkg{t, Logger{}}

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

	if !strings.Contains(errOut, "Expected image flag when pulling a image or index, please use --image instead of -b") {
		t.Fatalf("Expected error to contain message about using the wrong pull flag, got: %s", errOut)
	}
}

func createBundleDir(dir string) (string, error) {
	imgpkgDir := filepath.Join(dir, ".imgpkg")
	err := os.Mkdir(imgpkgDir, 0700)
	if err != nil {
		return "", err
	}

	fileContents := map[string]string{
		"bundle.yml": bundleYAML,
		"images.yml": imagesYAML,
	}
	for filename, contents := range fileContents {
		err = ioutil.WriteFile(filepath.Join(imgpkgDir, filename), []byte(contents), 0600)
		if err != nil {
			return imgpkgDir, err
		}
	}
	return imgpkgDir, nil
}
