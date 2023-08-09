// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

func TestNewContentsBundleWithBundles(t *testing.T) {
	imagesLockYAML := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: my.registry.io/bundle@sha256:703218c0465075f4425e58fac086e09e1de5c340b12976ab9eb8ad26615c3715
`
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

		_, err = subject.Push(imgTag, fakeRegistry, util.NewNoopLevelLogger())
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

		_, err = subject.Push(imgTag, fakeRegistry, util.NewNoopLevelLogger())
		if err != nil {
			t.Fatalf("not expecting push to fail: %s", err)
		}
	})
}
