// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	ctlbundle "github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/image/imagefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestImagesLock_WriteToPath(t *testing.T) {
	assets := helpers.Assets{T: t}
	defer assets.CleanCreatedFolders()

	t.Run("When an Image is not a Bundle, it does not update the ImagesLock file", func(t *testing.T) {
		bundleFolder := assets.CreateTempFolder("no-update")
		require.NoError(t, os.MkdirAll(filepath.Join(bundleFolder, ".imgpkg"), 0700))

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
		require.NoError(t, err)
		assert.Contains(t, uiOutput, "skipping lock file update")

		resultImagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(bundleFolder, ".imgpkg", "images.yml"))
		require.NoError(t, err)
		require.Len(t, resultImagesLock.Images, 2)
		assert.Equal(t, imageLock.Images[0].Image, resultImagesLock.Images[0].Image)
		assert.Equal(t, imageLock.Images[1].Image, resultImagesLock.Images[1].Image)
	})

	t.Run("when all images are in the bundle repo, it updates the ImagesLock file", func(t *testing.T) {
		bundleFolder := assets.CreateTempFolder("updated")
		require.NoError(t, os.MkdirAll(filepath.Join(bundleFolder, ".imgpkg"), 0700))

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
		require.NoError(t, err)
		require.Contains(t, uiOutput, "hosting every image specified in the bundle's Images Lock file (.imgpkg/images.yml)")

		resultImagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(bundleFolder, ".imgpkg", "images.yml"))
		require.NoError(t, err)
		require.Len(t, resultImagesLock.Images, 1)
		assert.NotEqual(t, imageLock.Images[0].Image, resultImagesLock.Images[0].Image)
	})
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

		newImagesLock, skipped, err := subject.LocalizeImagesLock(true)
		require.NoError(t, err)
		assert.False(t, skipped)

		require.Len(t, newImagesLock.Images, 2)
		assert.Equal(t, "some.repo.io/bundle@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", newImagesLock.Images[0].Locations()[0])
		assert.Equal(t, "some.repo.io/bundle@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", newImagesLock.Images[1].Locations()[0])
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

		newImagesLock, skipped, err := subject.LocalizeImagesLock(true)
		require.NoError(t, err)
		assert.True(t, skipped)

		require.Len(t, newImagesLock.Images, 2)
		assert.Equal(t, "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", newImagesLock.Images[0].Locations()[0])
		assert.Equal(t, "some.repo.io/img2@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", newImagesLock.Images[1].Locations()[0])
	})
}

func TestImagesLock_LocationPrunedImagesLock(t *testing.T) {
	t.Run("when image cannot be found in primary location, it prunes that location from the list", func(t *testing.T) {
		imgRef := lockconfig.ImageRef{
			Image: "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
		}
		imgRef.AddLocation("second.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80")
		imgRef.AddLocation("first.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80")
		imagesLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				imgRef,
			},
		}
		fakeImagesMetadata := &imagefakes.FakeImagesMetadata{}

		// does not find image in first.repo.io/img1
		fakeImagesMetadata.DigestReturnsOnCall(0, regv1.Hash{}, errors.New("not found"))

		subject := ctlbundle.NewImagesLock(imagesLock, fakeImagesMetadata, "some.repo.io/bundle")
		imgRefs, err := subject.LocationPrunedImageRefs()
		require.NoError(t, err)
		assert.Equal(t, "second.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", imgRefs[0].PrimaryLocation())
	})

	t.Run("when image is found in primary location, it does not prune any location", func(t *testing.T) {
		imgRef := lockconfig.ImageRef{
			Image: "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
		}
		imgRef.AddLocation("second.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80")
		imgRef.AddLocation("first.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80")
		imagesLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				imgRef,
			},
		}
		fakeImagesMetadata := &imagefakes.FakeImagesMetadata{}

		subject := ctlbundle.NewImagesLock(imagesLock, fakeImagesMetadata, "some.repo.io/bundle")
		imgRefs, err := subject.LocationPrunedImageRefs()
		require.NoError(t, err)
		assert.Equal(t, "first.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", imgRefs[0].PrimaryLocation())
	})
}

func TestImagesLock_Merge(t *testing.T) {
	t.Run("appends the images from the provided ImagesLock", func(t *testing.T) {
		parentImagesLock := lockconfig.ImagesLock{
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
		subject := ctlbundle.NewImagesLock(parentImagesLock, fakeImagesMetadata, "some.repo.io/bundle")
		imgLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				{
					Image: "original.repo.io/img4@sha256:4322479b268761c699a2b8c09ac6877acdc17d8f2c1ce2a7f5ebc0a8ee754332",
				},
			},
		}
		imagesLockToMerge := ctlbundle.NewImagesLock(imgLock, fakeImagesMetadata, "some.repo.io/bundle")
		require.NoError(t, subject.Merge(imagesLockToMerge))

		imagesRefs := subject.ImageRefs()
		require.Len(t, imagesRefs, 3)
		require.Equal(t, "original.repo.io/img4@sha256:4322479b268761c699a2b8c09ac6877acdc17d8f2c1ce2a7f5ebc0a8ee754332", imagesRefs[2].Image)
		require.Len(t, imagesRefs[2].Locations(), 1)
		locations := imagesRefs[2].Locations()
		assert.Equal(t, "original.repo.io/img4@sha256:4322479b268761c699a2b8c09ac6877acdc17d8f2c1ce2a7f5ebc0a8ee754332", locations[0])
	})

	t.Run("when images are repeated ignores new image", func(t *testing.T) {
		parentImagesLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				{
					Image:       "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
					Annotations: map[string]string{"will be": "kept"},
				},
			},
		}
		fakeImagesMetadata := &imagefakes.FakeImagesMetadata{}
		subject := ctlbundle.NewImagesLock(parentImagesLock, fakeImagesMetadata, "some.repo.io/bundle")
		imgLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				{
					Image:       "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
					Annotations: map[string]string{"will not be": "added"},
				},
			},
		}
		imagesLockToMerge := ctlbundle.NewImagesLock(imgLock, fakeImagesMetadata, "some.repo.io/bundle")
		require.NoError(t, subject.Merge(imagesLockToMerge))

		imagesRefs := subject.ImageRefs()
		require.Len(t, imagesRefs, 1)
		assert.Equal(t, "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", imagesRefs[0].Image)
		assert.Equal(t, map[string]string{"will be": "kept"}, imagesRefs[0].Annotations)
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
