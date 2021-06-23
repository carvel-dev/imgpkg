// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"errors"
	"net/http"
	"os"
	"testing"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	ctlbundle "github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/image/imagefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestImagesLock_LocalizeImagesLock(t *testing.T) {
	config := &bundlefakes.FakeImagesLockLocationConfig{}
	config.FetchReturns(ctlbundle.ImageLocationsConfig{}, &ctlbundle.LocationsNotFound{})

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
		fakeImagesMetadata.ImageReturns(nil, &transport.Error{
			StatusCode: http.StatusNotFound,
		})
		fakeImagesMetadata.FirstImageExistsCalls(func(strings []string) (string, error) {
			return strings[0], nil
		})
		subject := ctlbundle.NewImagesLock(imagesLock, fakeImagesMetadata, "some.repo.io/bundle", config)

		bundleImgRefs, newImagesLock, skipped, err := subject.LocalizeImagesLock()
		require.NoError(t, err)
		assert.False(t, skipped)

		require.Len(t, newImagesLock.Images, 2)
		assert.Equal(t, "some.repo.io/bundle@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", newImagesLock.Images[0].PrimaryLocation())
		assert.Equal(t, "some.repo.io/bundle@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", newImagesLock.Images[1].PrimaryLocation())

		require.Len(t, bundleImgRefs.ImageRefs(), 2)
		assert.Equal(t, "some.repo.io/bundle@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", bundleImgRefs.ImageRefs()[0].PrimaryLocation())
		assert.Equal(t, "some.repo.io/bundle@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", bundleImgRefs.ImageRefs()[1].PrimaryLocation())
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
		fakeImagesMetadata.ImageReturns(nil, &transport.Error{
			StatusCode: http.StatusNotFound,
		})

		subject := ctlbundle.NewImagesLock(imagesLock, fakeImagesMetadata, "some.repo.io/bundle", config)

		// Other calls will return the default empty Hash and nil error
		fakeImagesMetadata.DigestReturnsOnCall(1, regv1.Hash{}, errors.New("not found"))

		bundleImgRefs, newImagesLock, skipped, err := subject.LocalizeImagesLock()
		require.NoError(t, err)
		assert.True(t, skipped)

		require.Len(t, newImagesLock.Images, 2)
		assert.Equal(t, "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", newImagesLock.Images[0].PrimaryLocation())
		assert.Equal(t, "some.repo.io/img2@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", newImagesLock.Images[1].PrimaryLocation())

		require.Len(t, bundleImgRefs.ImageRefs(), 2)
		assert.Equal(t, "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", bundleImgRefs.ImageRefs()[0].PrimaryLocation())
		assert.Equal(t, "some.repo.io/img2@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", bundleImgRefs.ImageRefs()[1].PrimaryLocation())
	})
}

func TestImagesLock_Merge(t *testing.T) {
	config := &bundlefakes.FakeImagesLockLocationConfig{}
	config.FetchReturns(ctlbundle.ImageLocationsConfig{}, &ctlbundle.LocationsNotFound{})

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
		subject := ctlbundle.NewImagesLock(parentImagesLock, fakeImagesMetadata, "some.repo.io/bundle", config)
		imgLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				{
					Image: "original.repo.io/img4@sha256:4322479b268761c699a2b8c09ac6877acdc17d8f2c1ce2a7f5ebc0a8ee754332",
				},
			},
		}
		imagesLockToMerge := ctlbundle.NewImagesLock(imgLock, fakeImagesMetadata, "some.repo.io/bundle", config)
		subject.Merge(imagesLockToMerge)

		imagesRefs, err := subject.ImageRefs()
		require.NoError(t, err)
		require.Len(t, imagesRefs.ImageRefs(), 3)
		require.Equal(t, "original.repo.io/img4@sha256:4322479b268761c699a2b8c09ac6877acdc17d8f2c1ce2a7f5ebc0a8ee754332", imagesRefs.ImageRefs()[2].Image)
		expectedLocations := []string{
			"some.repo.io/bundle@sha256:4322479b268761c699a2b8c09ac6877acdc17d8f2c1ce2a7f5ebc0a8ee754332",
			"original.repo.io/img4@sha256:4322479b268761c699a2b8c09ac6877acdc17d8f2c1ce2a7f5ebc0a8ee754332",
		}
		require.Equal(t, expectedLocations, imagesRefs.ImageRefs()[2].Locations())
	})

	t.Run("when images are repeated replaces with new image", func(t *testing.T) {
		parentImagesLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				{
					Image:       "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
					Annotations: map[string]string{"will be": "removed"},
				},
			},
		}
		fakeImagesMetadata := &imagefakes.FakeImagesMetadata{}
		subject := ctlbundle.NewImagesLock(parentImagesLock, fakeImagesMetadata, "some.repo.io/bundle", config)
		imgLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				{
					Image:       "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
					Annotations: map[string]string{"will be": "added"},
				},
			},
		}
		imagesLockToMerge := ctlbundle.NewImagesLock(imgLock, fakeImagesMetadata, "some.repo.io/bundle", nil)
		subject.Merge(imagesLockToMerge)

		imagesRefs, err := subject.ImageRefs()
		require.NoError(t, err)
		require.Len(t, imagesRefs.ImageRefs(), 1)
		assert.Equal(t, "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", imagesRefs.ImageRefs()[0].Image)
		assert.Equal(t, map[string]string{"will be": "added"}, imagesRefs.ImageRefs()[0].Annotations)
	})
}
