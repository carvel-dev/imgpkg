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
		subject := ctlbundle.NewImageRefsFromLock(imagesLock)

		imageRefs, err := subject.LocalizeAndFindImages(fakeImagesMetadata, config, "some.repo.io/bundle")
		require.NoError(t, err)
		assert.True(t, imageRefs.CollocatedWithBundle())

		require.Len(t, imageRefs.ImageRefs(), 2)
		assert.Equal(t, "some.repo.io/bundle@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", imageRefs.ImageRefs()[0].PrimaryLocation())
		assert.Equal(t, "some.repo.io/bundle@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", imageRefs.ImageRefs()[1].PrimaryLocation())
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

		subject := ctlbundle.NewImageRefsFromLock(imagesLock)

		// Other calls will return the default empty Hash and nil error
		fakeImagesMetadata.DigestReturnsOnCall(1, regv1.Hash{}, errors.New("not found"))

		imageRefs, err := subject.LocalizeAndFindImages(fakeImagesMetadata, config, "some.repo.io/bundle")
		require.NoError(t, err)
		assert.False(t, imageRefs.CollocatedWithBundle())

		require.Len(t, imageRefs.ImageRefs(), 2)
		assert.Equal(t, "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", imageRefs.ImageRefs()[0].PrimaryLocation())
		assert.Equal(t, "some.repo.io/img2@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", imageRefs.ImageRefs()[1].PrimaryLocation())
	})
}
