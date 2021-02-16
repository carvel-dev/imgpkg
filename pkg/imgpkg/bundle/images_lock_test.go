// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	ctlbundle "github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/image/imagefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestImagesLock_WriteToPath_WhenAnImageIsNotInBundleRepo_DoesNotUpdateTheImagesInImagesLockFile(t *testing.T) {
	bundleFolder, err := createBundleFolder()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.RemoveAll(bundleFolder)

	imageLock := lockconfig.ImagesLock{
		LockVersion: lockconfig.LockVersion{
			APIVersion: lockconfig.ImagesLockAPIVersion,
			Kind:       lockconfig.ImagesLockKind,
		},
		Images: []lockconfig.ImageRef{
			{
				Image: "some.place/repo@sha256:8136ff3a64517457b91f86bf66b8ffe13b986aaf3511887eda107e59dcb8c632",
			},
			{
				Image: "gcr.io/cf-k8s-lifecycle-tooling-klt/nginx@sha256:f35b49b1d18e083235015fd4bbeeabf6a49d9dc1d3a1f84b7df3794798b70c13",
			},
		},
	}
	fakeDigestRetrieval := func(reference regname.Reference) (regv1.Hash, error) {
		// Error out when checking for nginx image in the same repository as the bundle
		if reference.Identifier() != "sha256:8136ff3a64517457b91f86bf66b8ffe13b986aaf3511887eda107e59dcb8c632" &&
			reference.Context().Name() == "some.place/repo" {
			return regv1.Hash{}, fmt.Errorf("failed")
		}
		return regv1.Hash{}, nil
	}
	uiOutput, err := runWriteToPath(imageLock, fakeDigestRetrieval, bundleFolder)
	if err != nil {
		t.Fatalf("writing the localized images.yml: %s", err)
	}
	if !strings.Contains(uiOutput, "skipping lock file update") {
		t.Fatalf("expected copy.\nExpected: 'skipping lock file update'\nGot:%s", uiOutput)
	}

	resultImagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(bundleFolder, ".imgpkg", "images.yml"))
	if err != nil {
		t.Fatalf("unable to read images lock file: %s", err)
	}
	if len(imageLock.Images) != len(resultImagesLock.Images) {
		t.Fatalf("expected to have same 2 images but had %d",
			len(resultImagesLock.Images),
		)
	}
	if imageLock.Images[0].Image != resultImagesLock.Images[0].Image {
		t.Fatalf("expected first image not to change but was changed to: %s",
			resultImagesLock.Images[0].Image,
		)
	}
	if imageLock.Images[1].Image != resultImagesLock.Images[1].Image {
		t.Fatalf("expected second image not to change but was changed to: %s",
			resultImagesLock.Images[1].Image,
		)
	}
}

func TestImagesLock_WriteToPath_WhenAllImagesAreInBundleRepo_UpdatesTheImagesInImagesLockFile(t *testing.T) {
	bundleFolder, err := createBundleFolder()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.RemoveAll(bundleFolder)

	imageLock := lockconfig.ImagesLock{
		LockVersion: lockconfig.LockVersion{
			APIVersion: lockconfig.ImagesLockAPIVersion,
			Kind:       lockconfig.ImagesLockKind,
		},
		Images: []lockconfig.ImageRef{
			{
				Image: "some.other.place/repo@sha256:8136ff3a64517457b91f86bf66b8ffe13b986aaf3511887eda107e59dcb8c632",
			},
		},
	}
	fakeDigestRetrieval := func(reference regname.Reference) (regv1.Hash, error) {
		if reference.Context().Name() != "some.place/repo" {
			return regv1.Hash{}, fmt.Errorf("not found")
		}
		return regv1.Hash{}, nil
	}
	uiOutput, err := runWriteToPath(imageLock, fakeDigestRetrieval, bundleFolder)
	if err != nil {
		t.Fatalf("writing the localized images.yml: %s", err)
	}
	if !strings.Contains(uiOutput, "Updating all images in the ImagesLock file") {
		t.Fatalf("did not print expected copy.\nExpected: 'Updating all images in the ImagesLock file'\nGot:%s", uiOutput)
	}

	resultImagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(bundleFolder, ".imgpkg", "images.yml"))
	if err != nil {
		t.Fatalf("unable to read images lock file: %s", err)
	}
	if len(imageLock.Images) != len(resultImagesLock.Images) {
		t.Fatalf("expected to have same 1 images but had %d",
			len(resultImagesLock.Images),
		)
	}
	if imageLock.Images[0].Image == resultImagesLock.Images[0].Image {
		t.Fatalf("expected image to have changed but did not")
	}
}

func TestImagesLock_LocalizeImagesLock(t *testing.T) {
	t.Run("When All images can be found in the bundle repository, it returns the new image location and skipped == false", func(t *testing.T) {
		imagesLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				{
					Image: "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
				},
				{
					Image: "some.repo.io/img2@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a",
				},
			},
		}
		fakeImagesMetadata := &imagefakes.FakeImagesMetadata{}
		subject := ctlbundle.NewImagesLock(imagesLock, fakeImagesMetadata, "some.repo.io/bundle")

		newImagesLock, skipped, err := subject.LocalizeImagesLock()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if skipped {
			t.Fatalf("not expecting the localization to be skipped")
		}

		if len(newImagesLock.Images) != 2 {
			t.Fatalf("unexpected number of images\nExpected: 2\nGot:%d", len(newImagesLock.Images))
		}
		if newImagesLock.Images[0].Image != "some.repo.io/bundle@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80" {
			t.Fatalf("unexpected image\nExpected: %s\nGot:%s", "some.repo.io/bundle@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", newImagesLock.Images[0].Image)
		}
		if newImagesLock.Images[1].Image != "some.repo.io/bundle@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a" {
			t.Fatalf("unexpected image\nExpected: %s\nGot:%s", "some.repo.io/bundle@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", newImagesLock.Images[1].Image)
		}
	})

	t.Run("When one image cannot be found in the bundle repository, it returns the old image location and skipped == true", func(t *testing.T) {
		imagesLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				{
					Image: "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
				},
				{
					Image: "some.repo.io/img2@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a",
				},
			},
		}
		fakeImagesMetadata := &imagefakes.FakeImagesMetadata{}
		subject := ctlbundle.NewImagesLock(imagesLock, fakeImagesMetadata, "some.repo.io/bundle")

		// Other calls will return the default empty Hash and nil error
		fakeImagesMetadata.DigestReturnsOnCall(1, regv1.Hash{}, errors.New("not found"))

		newImagesLock, skipped, err := subject.LocalizeImagesLock()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if !skipped {
			t.Fatalf("expecting the localization to be skipped, but it was not")
		}

		if len(newImagesLock.Images) != 2 {
			t.Fatalf("unexpected number of images\nExpected: 2\nGot:%d", len(newImagesLock.Images))
		}
		if newImagesLock.Images[0].Image != "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80" {
			t.Fatalf("unexpected image\nExpected: %s\nGot:%s", "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", newImagesLock.Images[0].Image)
		}
		if newImagesLock.Images[1].Image != "some.repo.io/img2@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a" {
			t.Fatalf("unexpected image\nExpected: %s\nGot:%s", "some.repo.io/img2@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", newImagesLock.Images[1].Image)
		}
	})
}

func runWriteToPath(imagesLock lockconfig.ImagesLock, fakeDigestHandler func(reference regname.Reference) (regv1.Hash, error), bundleFolder string) (string, error) {
	fakeRegistry := &imagefakes.FakeImagesMetadata{}
	fakeRegistry.DigestCalls(fakeDigestHandler)
	uiOutput := ""
	uiFake := &bundlefakes.FakeUI{}
	uiFake.BeginLinefCalls(func(s string, i ...interface{}) {
		uiOutput = fmt.Sprintf("%s%s", uiOutput, fmt.Sprintf(s, i...))
	})
	subject := ctlbundle.NewImagesLock(imagesLock, fakeRegistry, "some.place/repo")
	return uiOutput, subject.WriteToPath(bundleFolder, uiFake)
}

func createBundleFolder() (string, error) {
	bundleFolder := filepath.Join(os.TempDir(), "images-lock-writer-test")
	err := os.MkdirAll(filepath.Join(bundleFolder, ".imgpkg"), 0700)
	if err != nil {
		return "", fmt.Errorf("unable to create folder: %s", err)
	}

	return bundleFolder, err
}
