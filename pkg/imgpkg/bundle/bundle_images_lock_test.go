// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundle_AllImagesLock(t *testing.T) {
	fakeRegistry := helpers.NewFakeRegistry(t)
	defer fakeRegistry.CleanUp()
	logger := &helpers.Logger{}
	img1 := fakeRegistry.WithRandomImage("library/img1")
	img2 := fakeRegistry.WithRandomImage("library/img2")
	bundle1 := fakeRegistry.WithRandomBundle("library/bundle1")
	bundle2 := fakeRegistry.WithRandomBundle("library/bundle2")

	t.Run("when a bundle contains only images it returns 2 locations for each image", func(t *testing.T) {
		fakeImagesLockReader := &bundlefakes.FakeImagesLockReader{}
		logger.Section("bundle1 contains 2 images", func() {
			bundle1ImagesLock := lockconfig.ImagesLock{
				Images: []lockconfig.ImageRef{
					{
						Image: img1.RefDigest,
					},
					{
						Image: img2.RefDigest,
					},
				},
			}
			fakeImagesLockReader.ReadReturns(bundle1ImagesLock, nil)
		})

		subject := bundle.NewBundleWithReader(bundle1.RefDigest, fakeRegistry.Build(), fakeImagesLockReader)
		resultImagesLock, err := subject.AllImagesLock(1)
		require.NoError(t, err)

		require.Equal(t, 1, fakeImagesLockReader.ReadCallCount())
		imageRefs := resultImagesLock.ImageRefs()
		require.NoError(t, err)
		require.Len(t, imageRefs, 2)
		logger.Section("check no locations where added", func() {
			img1Digest, err := regname.NewDigest(img1.RefDigest)
			require.NoError(t, err)
			assert.Equal(t, []string{
				fakeRegistry.ReferenceOnTestServer("library/bundle1@" + img1Digest.DigestStr()),
				img1.RefDigest,
			}, imageRefs[0].Locations(), "expects bundle1 repository and original location")

			img2Digest, err := regname.NewDigest(img2.RefDigest)
			require.NoError(t, err)
			assert.Equal(t, []string{
				fakeRegistry.ReferenceOnTestServer("library/bundle1@" + img2Digest.DigestStr()),
				img2.RefDigest,
			}, imageRefs[1].Locations(), "expects bundle1 repository and original location")
		})
	})

	t.Run("when bundle2 contains a nested bundle and bundle2 it returns 3 possible locations for each image", func(t *testing.T) {
		fakeImagesLockReader := &bundlefakes.FakeImagesLockReader{}

		logger.Section("bundle1 contains 2 images", func() {
			bundle2ImagesLock := lockconfig.ImagesLock{
				Images: []lockconfig.ImageRef{
					{
						Image: img1.RefDigest,
					},
					{
						Image: img2.RefDigest,
					},
				},
			}
			fakeImagesLockReader.ReadReturnsOnCall(1, bundle2ImagesLock, nil)
		})

		logger.Section("bundle2 contains only bundle1", func() {
			bundle2ImagesLock := lockconfig.ImagesLock{
				Images: []lockconfig.ImageRef{
					{
						Image: bundle1.RefDigest,
					},
				},
			}
			fakeImagesLockReader.ReadReturnsOnCall(0, bundle2ImagesLock, nil)
		})

		subject := bundle.NewBundleWithReader(bundle2.RefDigest, fakeRegistry.Build(), fakeImagesLockReader)
		resultImagesLock, err := subject.AllImagesLock(1)
		require.NoError(t, err)

		require.Equal(t, 2, fakeImagesLockReader.ReadCallCount())
		imgRefs := resultImagesLock.ImageRefs()
		require.Len(t, imgRefs, 3)

		logger.Section("check locations are present for all images", func() {
			bundle1Digest, err := regname.NewDigest(bundle1.RefDigest)
			require.NoError(t, err)

			assert.Equal(t, []string{
				fakeRegistry.ReferenceOnTestServer("library/bundle2@" + bundle1Digest.DigestStr()),
				bundle1.RefDigest,
			}, imgRefs[0].Locations(), "expects bundle2 repository and original bundle location")

			img1Digest, err := regname.NewDigest(img1.RefDigest)
			require.NoError(t, err)
			assert.Equal(t, []string{
				fakeRegistry.ReferenceOnTestServer("library/bundle2@" + img1Digest.DigestStr()),
				fakeRegistry.ReferenceOnTestServer("library/bundle1@" + img1Digest.DigestStr()),
				img1.RefDigest,
			}, imgRefs[1].Locations(), "expects bundle2 repository, bundle1 repository and original image location")

			img2Digest, err := regname.NewDigest(img2.RefDigest)
			require.NoError(t, err)
			assert.Equal(t, []string{
				fakeRegistry.ReferenceOnTestServer("library/bundle2@" + img2Digest.DigestStr()),
				fakeRegistry.ReferenceOnTestServer("library/bundle1@" + img2Digest.DigestStr()),
				img2.RefDigest,
			}, imgRefs[2].Locations(), "expects bundle2 repository, bundle1 repository and original image location")
		})
	})

	t.Run("when a nested bundle is present twice it only returns each image once", func(t *testing.T) {
		fakeImagesLockReader := &bundlefakes.FakeImagesLockReader{}
		bundle3 := fakeRegistry.WithRandomBundle("library/bundle3")

		logger.Section("bundle1 contains 2 images", func() {
			bundle2ImagesLock := lockconfig.ImagesLock{
				Images: []lockconfig.ImageRef{
					{
						Image: img1.RefDigest,
					},
					{
						Image: img2.RefDigest,
					},
				},
			}
			fakeImagesLockReader.ReadReturnsOnCall(2, bundle2ImagesLock, nil)
		})

		logger.Section("bundle2 contains only bundle1", func() {
			bundle2ImagesLock := lockconfig.ImagesLock{
				Images: []lockconfig.ImageRef{
					{
						Image: bundle1.RefDigest,
					},
				},
			}
			fakeImagesLockReader.ReadReturnsOnCall(1, bundle2ImagesLock, nil)
		})

		logger.Section("bundle3 contains only bundle2 and bundle1", func() {
			bundle2ImagesLock := lockconfig.ImagesLock{
				Images: []lockconfig.ImageRef{
					{
						Image: bundle2.RefDigest,
					},
					{
						Image: bundle1.RefDigest,
					},
				},
			}
			fakeImagesLockReader.ReadReturnsOnCall(0, bundle2ImagesLock, nil)
		})

		fakeImagesReaderWriter := fakeRegistry.Build()
		subject := bundle.NewBundleWithReader(bundle3.RefDigest, fakeImagesReaderWriter, fakeImagesLockReader)
		resultImagesLock, err := subject.AllImagesLock(1)
		require.NoError(t, err)

		require.Equal(t, 3, fakeImagesLockReader.ReadCallCount())
		imgRefs := resultImagesLock.ImageRefs()
		require.NoError(t, err)

		logger.Section("check locations are present for all images", func() {
			bundle2Digest, err := regname.NewDigest(bundle2.RefDigest)
			require.NoError(t, err)
			assert.Equal(t, []string{
				fakeRegistry.ReferenceOnTestServer("library/bundle3@" + bundle2Digest.DigestStr()),
				bundle2.RefDigest,
			}, imgRefs[0].Locations(), "expects bundle3 repository and original bundle location")

			bundle1Digest, err := regname.NewDigest(bundle1.RefDigest)
			require.NoError(t, err)
			assert.Equal(t, []string{
				fakeRegistry.ReferenceOnTestServer("library/bundle3@" + bundle1Digest.DigestStr()),
				bundle1.RefDigest,
			}, imgRefs[1].Locations(), "expects bundle3 repository and original bundle location")

			img1Digest, err := regname.NewDigest(img1.RefDigest)
			require.NoError(t, err)
			assert.Equal(t, []string{
				fakeRegistry.ReferenceOnTestServer("library/bundle3@" + img1Digest.DigestStr()),
				fakeRegistry.ReferenceOnTestServer("library/bundle2@" + img1Digest.DigestStr()),
				img1.RefDigest,
			}, imgRefs[2].Locations(), "expects bundle3, bundle2 repository, and original image location")

			img2Digest, err := regname.NewDigest(img2.RefDigest)
			require.NoError(t, err)
			assert.Equal(t, []string{
				fakeRegistry.ReferenceOnTestServer("library/bundle3@" + img2Digest.DigestStr()),
				fakeRegistry.ReferenceOnTestServer("library/bundle2@" + img2Digest.DigestStr()),
				img2.RefDigest,
			}, imgRefs[3].Locations(), "expects bundle3, bundle2 repository, and original image location")
		})
	})

	t.Run("when a nested bundle is present twice it only checks the registry once per image", func(t *testing.T) {
		fakeImagesLockReader := &bundlefakes.FakeImagesLockReader{}
		bundle3 := fakeRegistry.WithRandomBundle("library/bundle3")

		logger.Section("bundle1 contains 2 images", func() {
			bundle2ImagesLock := lockconfig.ImagesLock{
				Images: []lockconfig.ImageRef{
					{
						Image: img1.RefDigest,
					},
					{
						Image: img2.RefDigest,
					},
				},
			}
			fakeImagesLockReader.ReadReturnsOnCall(2, bundle2ImagesLock, nil)
		})

		logger.Section("bundle2 contains only bundle1", func() {
			bundle2ImagesLock := lockconfig.ImagesLock{
				Images: []lockconfig.ImageRef{
					{
						Image: bundle1.RefDigest,
					},
				},
			}
			fakeImagesLockReader.ReadReturnsOnCall(1, bundle2ImagesLock, nil)
		})

		logger.Section("bundle3 contains only bundle2 and bundle1", func() {
			bundle2ImagesLock := lockconfig.ImagesLock{
				Images: []lockconfig.ImageRef{
					{
						Image: bundle2.RefDigest,
					},
					{
						Image: bundle1.RefDigest,
					},
				},
			}
			fakeImagesLockReader.ReadReturnsOnCall(0, bundle2ImagesLock, nil)
		})

		fakeImagesReaderWriter := fakeRegistry.Build()
		subject := bundle.NewBundleWithReader(bundle3.RefDigest, fakeImagesReaderWriter, fakeImagesLockReader)
		_, err := subject.AllImagesLock(1)
		require.NoError(t, err)

		require.Equal(t, 3, fakeImagesLockReader.ReadCallCount())
	})
}
