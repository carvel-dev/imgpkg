// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"net/http"
	"os"
	"testing"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctlbundle "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset/imagesetfakes"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestImagesRefs_LocalizeImagesLock(t *testing.T) {
	config := &bundlefakes.FakeImagesLockLocationConfig{}
	config.FetchReturns(ctlbundle.ImageLocationsConfig{}, &ctlbundle.LocationsNotFound{})

	t.Run("When All images can be found in the bundle repository, it returns the new image location and colocated == true", func(t *testing.T) {
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

		fakeImagesMetadata := &imagesetfakes.FakeImagesMetadata{}
		fakeImagesMetadata.ImageReturns(nil, &transport.Error{
			StatusCode: http.StatusNotFound,
		})
		fakeImagesMetadata.FirstImageExistsCalls(func(strings []string) (string, error) {
			return strings[0], nil
		})
		subject, err := ctlbundle.NewImageRefsFromImagesLock(imagesLock, config)
		require.NoError(t, err)

		colocated, err := subject.UpdateRelativeToRepo(fakeImagesMetadata, "some.repo.io/bundle")
		require.NoError(t, err)
		assert.True(t, colocated)

		newImagesLock := subject.ImagesLock()
		require.Len(t, newImagesLock.Images, 2)
		assert.Equal(t, "some.repo.io/bundle@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", newImagesLock.Images[0].PrimaryLocation())
		assert.Equal(t, "some.repo.io/bundle@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", newImagesLock.Images[1].PrimaryLocation())

		require.Len(t, subject.ImageRefs(), 2)
		assert.Equal(t, "some.repo.io/bundle@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", subject.ImageRefs()[0].PrimaryLocation())
		assert.Equal(t, "some.repo.io/bundle@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", subject.ImageRefs()[1].PrimaryLocation())
	})

	t.Run("When one image cannot be found in the bundle repository, it returns the old image location and colocated == false", func(t *testing.T) {
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
		fakeImagesMetadata := &imagesetfakes.FakeImagesMetadata{}
		fakeImagesMetadata.ImageReturns(nil, &transport.Error{
			StatusCode: http.StatusNotFound,
		})

		subject, err := ctlbundle.NewImageRefsFromImagesLock(imagesLock, config)
		require.NoError(t, err)

		// Other calls will return the default empty Hash and nil error
		fakeImagesMetadata.DigestReturnsOnCall(1, regv1.Hash{}, &transport.Error{
			StatusCode: http.StatusNotFound,
		})

		colocated, err := subject.UpdateRelativeToRepo(fakeImagesMetadata, "some.repo.io/bundle")
		require.NoError(t, err)
		assert.False(t, colocated)

		newImagesLock := subject.ImagesLock()
		require.Len(t, newImagesLock.Images, 2)
		assert.Equal(t, "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", newImagesLock.Images[0].PrimaryLocation())
		assert.Equal(t, "some.repo.io/img2@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", newImagesLock.Images[1].PrimaryLocation())

		require.Len(t, subject.ImageRefs(), 2)
		assert.Equal(t, "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", subject.ImageRefs()[0].PrimaryLocation())
		assert.Equal(t, "some.repo.io/img2@sha256:45f3926bca9fc42adb650fef2a41250d77841dde49afc8adc7c0c633b3d5f27a", subject.ImageRefs()[1].PrimaryLocation())
	})

	t.Run("When one images is present twice without locations OCI Image, it returns the ImagesLock with both images", func(t *testing.T) {
		imagesLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				{
					Image: "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
					Annotations: map[string]string{
						"annot": "1",
					},
				},
				{
					Image: "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
					Annotations: map[string]string{
						"annot": "2",
					},
				},
			},
		}
		fakeImagesMetadata := &imagesetfakes.FakeImagesMetadata{}
		fakeImagesMetadata.ImageReturns(nil, &transport.Error{
			StatusCode: http.StatusNotFound,
		})

		subject, err := ctlbundle.NewImageRefsFromImagesLock(imagesLock, config)
		require.NoError(t, err)

		// Other calls will return the default empty Hash and nil error
		fakeImagesMetadata.DigestReturnsOnCall(0, regv1.Hash{}, &transport.Error{
			StatusCode: http.StatusNotFound,
		})

		updatedRelativeToRepo, err := subject.UpdateRelativeToRepo(fakeImagesMetadata, "some.repo.io/bundle")
		require.NoError(t, err)
		require.False(t, updatedRelativeToRepo)

		require.Len(t, subject.ImagesLock().Images, 2)
		assert.Equal(t, "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", subject.ImagesLock().Images[0].PrimaryLocation())
		assert.Equal(t, "1", subject.ImagesLock().Images[0].Annotations["annot"])
		assert.Equal(t, "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", subject.ImagesLock().Images[1].PrimaryLocation())
		assert.Equal(t, "2", subject.ImagesLock().Images[1].Annotations["annot"])
	})

	t.Run("When one images is present twice with locations OCI Image, it returns the ImagesLock with both images", func(t *testing.T) {
		imagesLock := lockconfig.ImagesLock{
			Images: []lockconfig.ImageRef{
				{
					Image: "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
					Annotations: map[string]string{
						"annot": "1",
					},
				},
				{
					Image: "some.repo.io/img1@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80",
					Annotations: map[string]string{
						"annot": "2",
					},
				},
			},
		}
		fakeImagesMetadata := &imagesetfakes.FakeImagesMetadata{}
		fakeImagesMetadata.ImageReturns(nil, &transport.Error{
			StatusCode: http.StatusOK,
		})
		config.FetchReturns(ctlbundle.ImageLocationsConfig{}, nil)

		subject, err := ctlbundle.NewImageRefsFromImagesLock(imagesLock, config)
		require.NoError(t, err)

		// Other calls will return the default empty Hash and nil error
		fakeImagesMetadata.DigestReturnsOnCall(0, regv1.Hash{}, &transport.Error{
			StatusCode: http.StatusNotFound,
		})

		updatedRelativeToRepo, err := subject.UpdateRelativeToRepo(fakeImagesMetadata, "some.repo.io/bundle")
		require.NoError(t, err)
		require.True(t, updatedRelativeToRepo)

		require.Len(t, subject.ImagesLock().Images, 2)
		assert.Equal(t, "some.repo.io/bundle@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", subject.ImagesLock().Images[0].PrimaryLocation())
		assert.Equal(t, "1", subject.ImagesLock().Images[0].Annotations["annot"])
		assert.Equal(t, "some.repo.io/bundle@sha256:27fde5fa39e3c97cb1e5dabfb664784b605a592d5d2df5482d744742efebba80", subject.ImagesLock().Images[1].PrimaryLocation())
		assert.Equal(t, "2", subject.ImagesLock().Images[1].Annotations["annot"])
	})
}
