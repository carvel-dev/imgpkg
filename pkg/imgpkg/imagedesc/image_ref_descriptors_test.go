// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package imagedesc_test

import (
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/imagedesc"
	"github.com/stretchr/testify/require"
)

func TestNewImageRefDescriptorsFromBytes(t *testing.T) {
	t.Run("rejects an image descriptor with no refs", func(t *testing.T) {
		manifest := `[{"Image":{"Refs":[],"Manifest":{"Digest":"sha256:aaaa"}}}]`

		_, err := imagedesc.NewImageRefDescriptorsFromBytes([]byte(manifest))
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one ref")
	})

	t.Run("rejects an image nested in an index with no refs", func(t *testing.T) {
		manifest := `[{"ImageIndex":{"Refs":["registry.example.com/repo@sha256:bbbb"],"Digest":"sha256:bbbb","Images":[{"Refs":[],"Manifest":{"Digest":"sha256:aaaa"}}]}}]`

		_, err := imagedesc.NewImageRefDescriptorsFromBytes([]byte(manifest))
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one ref")
	})

	t.Run("accepts a descriptor with a ref", func(t *testing.T) {
		manifest := `[{"Image":{"Refs":["registry.example.com/repo@sha256:aaaa"],"Manifest":{"Digest":"sha256:aaaa"}}}]`

		ids, err := imagedesc.NewImageRefDescriptorsFromBytes([]byte(manifest))
		require.NoError(t, err)
		require.Len(t, ids.Descriptors(), 1)
	})
}
