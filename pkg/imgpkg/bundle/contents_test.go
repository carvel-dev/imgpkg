// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/require"
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
	assets := &helpers.Assets{T: t}
	defer assets.CleanCreatedFolders()
	bundleBuilder := helpers.NewBundleDir(t, assets)
	bundleDir := bundleBuilder.CreateBundleDir(helpers.BundleYAML, imagesLockYAML)

	bundleImg := &fake.FakeImage{}
	cfgFile := &v1.ConfigFile{
		Config: v1.Config{
			Labels: map[string]string{"dev.carvel.imgpkg.bundle": "true"},
		},
	}
	bundleImg.ConfigFileReturns(cfgFile, nil)
	fakeRegistry.ImageReturns(bundleImg, nil)

	t.Run("push is successful", func(t *testing.T) {
		subject := bundle.NewContents([]string{bundleDir}, nil)
		imgTag, err := name.NewTag("my.registry.io/new-bundle:tag")
		if err != nil {
			t.Fatalf("failed to read tag: %s", err)
		}

		_, err = subject.Push(imgTag, fakeRegistry, fakeUI)
		if err != nil {
			t.Fatalf("not expecting push to fail: %s", err)
		}
	})

	t.Run("build is successful", func(t *testing.T) {
		subject := bundle.NewContents([]string{bundleDir}, nil)

		fileImage, err := subject.Build(fakeUI)
		require.NoError(t, err)
		config, err := fileImage.ConfigFile()
		require.NoError(t, err)
		require.Contains(t, config.Config.Labels, "dev.carvel.imgpkg.bundle")
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
	assets := &helpers.Assets{T: t}
	defer assets.CleanCreatedFolders()
	bundleBuilder := helpers.NewBundleDir(t, assets)
	bundleDir := bundleBuilder.CreateBundleDir(helpers.BundleYAML, imagesLockYAML)

	bundleImg := &fake.FakeImage{}
	cfgFile := &v1.ConfigFile{
		Config: v1.Config{},
	}
	bundleImg.ConfigFileReturns(cfgFile, nil)
	fakeRegistry.ImageReturns(bundleImg, nil)

	t.Run("push is successful", func(t *testing.T) {
		subject := bundle.NewContents([]string{bundleDir}, nil)
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
