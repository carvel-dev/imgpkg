// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
)

func TestNewContentsBundleWithBundles(t *testing.T) {
	imagesLockYAML := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: my.registry.io/bundle@sha256:703218c0465075f4425e58fac086e09e1de5c340b12976ab9eb8ad26615c3715
`
	fakeUI := &bundlefakes.FakeUI{}
	fakeRegistry := &bundlefakes.FakeImagesMetadataWriter{}
	bundleDir := createBundleDir(t, "inner-bundle", imagesLockYAML)
	defer os.RemoveAll(bundleDir)
	bundleImg := &fake.FakeImage{}
	cfgFile := &v1.ConfigFile{
		Config: v1.Config{
			Labels: map[string]string{"dev.carvel.imgpkg.bundle": "true"},
		},
	}
	bundleImg.ConfigFileReturns(cfgFile, nil)
	fakeRegistry.ImageReturns(bundleImg, nil)

	t.Run("when allowInnerBundles is false, push fails", func(t *testing.T) {
		subject := bundle.NewContents([]string{bundleDir}, nil, false)
		imgTag, err := name.NewTag("my.registry.io/new-bundle:tag")
		if err != nil {
			t.Fatalf("failed to read tag: %s", err)
		}

		_, err = subject.Push(imgTag, fakeRegistry, fakeUI)
		if err == nil {
			t.Fatalf("expected test to fail, but it did not")
		}
		expectedError := "Expected image lock to not contain bundle reference: 'my.registry.io/bundle@sha256:703218c0465075f4425e58fac086e09e1de5c340b12976ab9eb8ad26615c3715'"
		if err.Error() != expectedError {
			t.Fatalf("Error message not expected\nExpected: %s\nGot: %s", expectedError, err.Error())
		}
	})

	t.Run("when allowInnerBundles is true, push is successful", func(t *testing.T) {
		subject := bundle.NewContents([]string{bundleDir}, nil, true)
		imgTag, err := name.NewTag("my.registry.io/new-bundle:tag")
		if err != nil {
			t.Fatalf("failed to read tag: %s", err)
		}

		_, err = subject.Push(imgTag, fakeRegistry, fakeUI)
		if err != nil {
			t.Fatalf("not expecting push to fail: %s", err)
		}
	})
}

func TestNewContentsBundleWithImages(t *testing.T) {
	imagesLockYAML := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: my.registry.io/image1@sha256:703218c0465075f4425e58fac086e09e1de5c340b12976ab9eb8ad26615c3715
`
	fakeUI := &bundlefakes.FakeUI{}
	fakeRegistry := &bundlefakes.FakeImagesMetadataWriter{}
	bundleDir := createBundleDir(t, "inner-bundle", imagesLockYAML)
	defer os.RemoveAll(bundleDir)
	bundleImg := &fake.FakeImage{}
	cfgFile := &v1.ConfigFile{
		Config: v1.Config{},
	}
	bundleImg.ConfigFileReturns(cfgFile, nil)
	fakeRegistry.ImageReturns(bundleImg, nil)

	t.Run("when allowInnerBundles is false, push is successful", func(t *testing.T) {
		subject := bundle.NewContents([]string{bundleDir}, nil, false)
		imgTag, err := name.NewTag("my.registry.io/new-bundle:tag")
		if err != nil {
			t.Fatalf("failed to read tag: %s", err)
		}

		_, err = subject.Push(imgTag, fakeRegistry, fakeUI)
		if err != nil {
			t.Fatalf("not expecting push to fail: %s", err)
		}
	})

	t.Run("when allowInnerBundles is true, push is successful", func(t *testing.T) {
		subject := bundle.NewContents([]string{bundleDir}, nil, true)
		imgTag, err := name.NewTag("my.registry.io/new-bundle:tag")
		if err != nil {
			t.Fatalf("failed to read tag: %s", err)
		}

		_, err = subject.Push(imgTag, fakeRegistry, fakeUI)
		if err != nil {
			t.Fatalf("not expecting push to fail: %s", err)
		}
	})
}

func createBundleDir(t *testing.T, prefix string, imagesYAML string) string {
	t.Helper()
	bundleDir, err := ioutil.TempDir("", prefix)
	if err != nil {
		t.Fatalf("unable to create bundle folder: %s", err)
	}

	imgpkgDir := filepath.Join(bundleDir, ".imgpkg")
	err = os.MkdirAll(imgpkgDir, 0700)
	if err != nil {
		t.Fatalf("unable to create imgpkg folder: %s", err)
	}

	err = ioutil.WriteFile(filepath.Join(imgpkgDir, "images.yml"), []byte(imagesYAML), 0600)
	if err != nil {
		t.Fatalf("unable to create images lock file: %s", err)
	}

	return bundleDir
}
