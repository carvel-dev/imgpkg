// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"carvel.dev/imgpkg/test/helpers"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const emptyImagesYaml = `apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
`

func TestMultiImgpkgDirError(t *testing.T) {
	tempDir := os.TempDir()

	pushDir := filepath.Join(tempDir, "imgpkg-push-units-multi-dir")
	defer Cleanup(pushDir)

	// cleaned up via pushDir
	fooDir := filepath.Join(pushDir, "foo")
	barDir := filepath.Join(pushDir, "bar")

	// cleanup any previous state
	Cleanup(pushDir)
	err := os.MkdirAll(fooDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = os.MkdirAll(barDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = createBundleDir(fooDir, "")
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = createBundleDir(barDir, "")
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{fooDir, barDir}}, BundleFlags: BundleFlags{Bundle: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), fmt.Sprintf("This directory contains multiple bundle definitions. Only a single instance of .imgpkg%simages.yml can be provided", string(os.PathSeparator))) {
		t.Fatalf("Expected error to contain message about multiple .imgpkg dirs, got: %s", err)
	}
}

func TestImageWithImgpkgDirError(t *testing.T) {
	tempDir := os.TempDir()

	pushDir := filepath.Join(tempDir, "imgpkg-push-units-image-dir")
	defer Cleanup(pushDir)

	// cleaned up via pushDir
	fooDir := filepath.Join(pushDir, "foo")
	err := os.MkdirAll(fooDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = createBundleDir(fooDir, "")
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{fooDir}}, ImageFlags: ImageFlags{Image: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	expected := "Images cannot be pushed with '.imgpkg' directories, consider using --bundle (-b) option"
	if err.Error() != expected {
		t.Fatalf("Expected error to contain message about image with bundle dir, got: %s", err)
	}
}

func TestNestedImgpkgDirError(t *testing.T) {
	tempDir := os.TempDir()
	pushDir := filepath.Join(tempDir, "imgpkg-push-units-nested-dir")
	defer Cleanup(pushDir)

	// cleaned up via push dir
	fooDir := filepath.Join(pushDir, "foo")
	err := os.MkdirAll(fooDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = createBundleDir(fooDir, "")
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	push := PushOptions{FileFlags: FileFlags{Files: []string{pushDir}}, BundleFlags: BundleFlags{Bundle: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected '.imgpkg' directory, to be a direct child of one of") {
		t.Fatalf("Expected error to contain message about .imgpkg being a direct child, got: %s", err)
	}
}

func TestBundleDirectoryErrors(t *testing.T) {
	testCases := []struct {
		name            string
		expectedError   string
		createBundleDir bool
	}{
		{
			name:            "bundle but no imagesLock",
			createBundleDir: true,
			expectedError:   "The bundle expected .imgpkg/images.yml to exist, but it wasn't found",
		},
		{
			name:            "no bundle",
			createBundleDir: false,
			expectedError:   fmt.Sprintf("This directory is not a bundle. It is missing .imgpkg%simages.yml", string(os.PathSeparator)),
		},
	}

	for _, tc := range testCases {
		f := func(t *testing.T) {
			tempDir := os.TempDir()
			pushDir := filepath.Join(tempDir, "imgpkg-push-dir")
			defer Cleanup(pushDir)

			err := os.Mkdir(pushDir, 0700)
			require.NoError(t, err)

			if tc.createBundleDir {
				err = createEmptyBundleDir(pushDir)
				require.NoError(t, err)
			}

			push := PushOptions{FileFlags: FileFlags{Files: []string{pushDir}}, BundleFlags: BundleFlags{Bundle: "foo"}}
			err = push.Run()
			require.Error(t, err)

			assert.Contains(t, err.Error(), tc.expectedError)
		}

		t.Run(tc.name, f)
	}
}

func TestDuplicateFilepathError(t *testing.T) {
	tempDir := os.TempDir()

	pushDir := filepath.Join(tempDir, "imgpkg-push-units-dup-filepath")
	defer Cleanup(pushDir)

	// cleaned up via pushDir
	fooDir := filepath.Join(pushDir, "foo")
	err := os.MkdirAll(fooDir, 0700)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	someFile := filepath.Join(fooDir, "some-file.yml")
	err = os.WriteFile(someFile, []byte("foo: bar"), 0600)
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	err = createBundleDir(fooDir, "")
	if err != nil {
		t.Fatalf("Failed to setup test: %s", err)
	}

	// duplicate someFile.yaml by including it directly and with the dir fooDir
	push := PushOptions{FileFlags: FileFlags{Files: []string{someFile, fooDir}}, BundleFlags: BundleFlags{Bundle: "foo"}}
	err = push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Found duplicate paths:") {
		t.Fatalf("Expected error to contain message about a duplicate filepath, got: %s", err)
	}
}

func TestNoImageOrBundleError(t *testing.T) {
	push := PushOptions{}
	err := push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected either image or bundle") {
		t.Fatalf("Expected error to contain message about invalid flags, got: %s", err)
	}
}

func TestImageAndBundleError(t *testing.T) {
	push := PushOptions{ImageFlags: ImageFlags{"image@123456"}, BundleFlags: BundleFlags{"my-bundle"}}
	err := push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Expected only one of image or bundle") {
		t.Fatalf("Expected error to contain message about invalid flags, got: %s", err)
	}
}

func TestImageAndBundleLockError(t *testing.T) {
	push := PushOptions{ImageFlags: ImageFlags{"image@123456"}, LockOutputFlags: LockOutputFlags{LockFilePath: "lock-file"}}
	err := push.Run()
	if err == nil {
		t.Fatalf("Expected validations to err, but did not")
	}

	if !strings.Contains(err.Error(), "Lock output is not compatible with image, use bundle for lock output") {
		t.Fatalf("Expected error to contain message about invalid flags, got: %s", err)
	}
}

func TestLabels(t *testing.T) {
	testCases := []struct {
		name           string
		opType         string
		expectedError  string
		expectedLabels map[string]string
		labelInput     string
	}{
		{
			name:           "bundle with multiple labels",
			opType:         "bundle",
			expectedError:  "",
			labelInput:     "foo=bar,bar=baz",
			expectedLabels: map[string]string{"dev.carvel.imgpkg.bundle": "true", "foo": "bar", "bar": "baz"},
		},
		{
			name:           "image with multiple labels",
			opType:         "image",
			expectedError:  "",
			labelInput:     "foo=bar,bar=baz",
			expectedLabels: map[string]string{"foo": "bar", "bar": "baz"},
		},
		{
			name:           "bundle with \".\" in label key",
			opType:         "bundle",
			expectedError:  "",
			labelInput:     "foo.bar=baz",
			expectedLabels: map[string]string{"dev.carvel.imgpkg.bundle": "true", "foo.bar": "baz"},
		},
		{
			name:           "bundle with long label key (> 64 chars)",
			opType:         "bundle",
			expectedError:  "",
			labelInput:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa=baz",
			expectedLabels: map[string]string{"dev.carvel.imgpkg.bundle": "true", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa": "baz"},
		},
		{
			name:           "bundle with long label value (> 256 chars)",
			opType:         "bundle",
			expectedError:  "",
			labelInput:     "foo=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expectedLabels: map[string]string{"dev.carvel.imgpkg.bundle": "true", "foo": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		},
		{
			name:           "bundle with spaces in label value",
			opType:         "bundle",
			expectedError:  "",
			labelInput:     "foo.bar=baz bar",
			expectedLabels: map[string]string{"dev.carvel.imgpkg.bundle": "true", "foo.bar": "baz bar"},
		},
	}

	for _, tc := range testCases {
		f := func(t *testing.T) {
			env := helpers.BuildEnv(t)
			imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
			defer env.Cleanup()

			opTypeFlag := "-b"
			pushDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)

			if tc.opType == "image" {
				opTypeFlag = "-i"
				pushDir = env.Assets.CreateAndCopySimpleApp("image-to-push")
			}

			if tc.labelInput == "" {
				imgpkg.Run([]string{"push", opTypeFlag, env.Image, "-f", pushDir})
			} else {
				imgpkg.Run([]string{"push", opTypeFlag, env.Image, "-l", tc.labelInput, "-f", pushDir})
			}

			ref, _ := name.NewTag(env.Image, name.WeakValidation)
			image, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
			require.NoError(t, err)

			config, err := image.ConfigFile()
			require.NoError(t, err)

			require.Equal(t, tc.expectedLabels, config.Config.Labels, "Expected labels provided via flags to match labels discovered on image")

		}

		t.Run(tc.name, f)
	}
}

func TestTags(t *testing.T) {
	testCases := []struct {
		name          string
		opType        string
		expectedError string
		expectedTags  []string
		tagInput      string
		inlineTag     string
	}{
		{
			name:          "bundle with one inline tag",
			opType:        "bundle",
			expectedError: "",
			tagInput:      "",
			expectedTags:  []string{"v1.0.1"},
			inlineTag:     "v1.0.1",
		},
		{
			name:          "bundle with one tag via flag",
			opType:        "bundle",
			expectedError: "",
			tagInput:      "v1.0.1",
			expectedTags:  []string{"v1.0.1", "latest"},
			inlineTag:     "",
		},
		{
			name:          "bundle with inline tag and tag via flag",
			opType:        "bundle",
			expectedError: "",
			tagInput:      "v1.2.0-alpha,latest",
			expectedTags:  []string{"v1.0.1", "v1.2.0-alpha", "latest"},
			inlineTag:     "v1.0.1",
		},
		{
			name:          "bundle with multiple tags via flag",
			opType:        "bundle",
			expectedError: "",
			tagInput:      "v1.0.1,v1.0.2",
			expectedTags:  []string{"v1.0.1", "v1.0.2", "latest"},
			inlineTag:     "",
		},
		{
			name:          "image with one inline tag",
			opType:        "image",
			expectedError: "",
			tagInput:      "",
			expectedTags:  []string{"v1.0.1"},
			inlineTag:     "v1.0.1",
		},
		{
			name:          "image with one tag via flag",
			opType:        "image",
			expectedError: "",
			tagInput:      "v1.0.1",
			expectedTags:  []string{"v1.0.1", "latest"},
			inlineTag:     "",
		},
		{
			name:          "image with inline tag and tags via flag",
			opType:        "image",
			expectedError: "",
			tagInput:      "latest,stable",
			expectedTags:  []string{"v1.0.1", "latest"},
			inlineTag:     "v1.0.1",
		},
	}

	for _, tc := range testCases {
		f := func(t *testing.T) {
			env := helpers.BuildEnv(t)
			targetImage := env.Image
			imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
			defer env.Cleanup()

			opTypeFlag := "-b"
			pushDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)

			if tc.opType == "image" {
				opTypeFlag = "-i"
				pushDir = env.Assets.CreateAndCopySimpleApp("image-to-push")
			}

			if tc.inlineTag != "" {
				targetImage = env.Image + ":" + tc.inlineTag
			}

			if tc.tagInput == "" {
				imgpkg.Run([]string{"push", opTypeFlag, targetImage, "-f", pushDir})
			} else {
				imgpkg.Run([]string{"push", opTypeFlag, targetImage, "--additional-tags", tc.tagInput, "-f", pushDir})
			}

			// Loop through expected tags and validate they exist on the image
			for _, tag := range tc.expectedTags {
				ref, _ := name.NewTag(env.Image+":"+tag, name.WeakValidation)
				image, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
				require.NoError(t, err)

				tagList := imgpkg.Run([]string{"tag", "ls", "-i", env.Image + ":" + tag})

				_, err = image.ConfigFile()
				require.NoError(t, err)

				require.Contains(t, tagList, tag, "Expected tags provided via flags to match tags discovered for image")

			}
		}

		t.Run(tc.name, f)
	}
}

func Cleanup(dirs ...string) {
	for _, dir := range dirs {
		os.RemoveAll(dir)
	}
}

func createBundleDir(loc, imagesYaml string) error {
	bundleDir := filepath.Join(loc, ".imgpkg")
	err := os.Mkdir(bundleDir, 0700)
	if err != nil {
		return err
	}

	if imagesYaml == "" {
		imagesYaml = emptyImagesYaml
	}
	return os.WriteFile(filepath.Join(bundleDir, "images.yml"), []byte(imagesYaml), 0600)
}

func createEmptyBundleDir(loc string) error {
	bundleDir := filepath.Join(loc, ".imgpkg")
	return os.Mkdir(bundleDir, 0700)
}
