// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/registry"
	v1 "carvel.dev/imgpkg/pkg/imgpkg/v1"
	"carvel.dev/imgpkg/test/helpers"
	"github.com/stretchr/testify/require"
)

func TestTagList(t *testing.T) {
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})

	img1 := fakeRegistry.WithRandomImage("some/image-1")
	img21 := fakeRegistry.WithRandomImage("some/image-2")
	img22 := fakeRegistry.WithRandomImage("some/image-2")
	// This image needs to be last because it will get the latest tag
	img2 := fakeRegistry.WithRandomImage("some/image-2")
	fakeRegistry.Tag(img21.RefDigest, "tag-2-1")
	fakeRegistry.Tag(img22.RefDigest, "tag-2-2")

	defer fakeRegistry.CleanUp()
	fakeRegistry.Build()

	t.Run("when only latest tag is present, it returns 1 tag", func(t *testing.T) {
		tagList, err := v1.TagList(img1.RefDigest, false, registry.Opts{})
		require.NoError(t, err)

		require.Equal(t, v1.TagsInfo{
			Repository: fakeRegistry.ReferenceOnTestServer("some/image-1"),
			Tags: []v1.TagInfo{
				{
					Tag:    "latest",
					Digest: "",
				},
			},
		}, tagList)
	})

	t.Run("when only latest tag is present and getDigests is set to true, it returns 1 tag and the associated digest", func(t *testing.T) {
		tagList, err := v1.TagList(img1.RefDigest, true, registry.Opts{})
		require.NoError(t, err)

		require.Equal(t, v1.TagsInfo{
			Repository: fakeRegistry.ReferenceOnTestServer("some/image-1"),
			Tags: []v1.TagInfo{
				{
					Tag:    "latest",
					Digest: img1.Digest,
				},
			},
		}, tagList)
	})

	t.Run("when multiple tags are present, it returns all tags", func(t *testing.T) {
		tagList, err := v1.TagList(img2.RefDigest, false, registry.Opts{})
		require.NoError(t, err)

		require.Equal(t, v1.TagsInfo{
			Repository: fakeRegistry.ReferenceOnTestServer("some/image-2"),
			Tags: []v1.TagInfo{
				{
					Tag:    "latest",
					Digest: "",
				},
				{
					Tag:    "tag-2-1",
					Digest: "",
				},
				{
					Tag:    "tag-2-2",
					Digest: "",
				},
			},
		}, tagList)
	})

	t.Run("when multiple tags are present and getDigests is set to true, it returns all tags and the associated digests", func(t *testing.T) {
		tagList, err := v1.TagList(img2.RefDigest, true, registry.Opts{})
		require.NoError(t, err)

		require.Equal(t, v1.TagsInfo{
			Repository: fakeRegistry.ReferenceOnTestServer("some/image-2"),
			Tags: []v1.TagInfo{
				{
					Tag:    "latest",
					Digest: img2.Digest,
				},
				{
					Tag:    "tag-2-1",
					Digest: img21.Digest,
				},
				{
					Tag:    "tag-2-2",
					Digest: img22.Digest,
				},
			},
		}, tagList)
	})
}
