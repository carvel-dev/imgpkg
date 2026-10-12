// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"runtime"
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/image"
	"github.com/stretchr/testify/require"
)

func TestTarImage(t *testing.T) {
	logger := testLogger{}
	// These digests include gzip output and were generated with Go 1.27.
	t.Run("Ensure image tar as the same SHA", func(t *testing.T) {
		tarImage := image.NewTarImage([]string{"test_assets/tar_folder"}, nil, logger, false)
		img, err := tarImage.AsFileImage(nil)
		require.NoError(t, err)
		d, err := img.Digest()
		require.NoError(t, err)
		if runtime.GOOS != "windows" {
			require.Equal(t, "sha256:100c793edd5b0690804b1a26806f1d8bc4879ddc9097428b72eed6746793f9a2", d.String())
		} else {
			require.Equal(t, "sha256:e63e15e0c2d861e95dcde54ef4d5828aaa391a852e074a379e50b46f2b265919", d.String())
		}
	})

	t.Run("When keeping the files and folder permissions ensure image tar as the same SHA", func(t *testing.T) {
		tarImage := image.NewTarImage([]string{"test_assets/tar_folder"}, nil, logger, true)
		img, err := tarImage.AsFileImage(nil)
		require.NoError(t, err)
		d, err := img.Digest()
		require.NoError(t, err)
		if runtime.GOOS != "windows" {
			require.Equal(t, "sha256:64c71c1b1ff6f57df5ad543e1a659953cf7e50912a39188ca866708186be1043", d.String())
		} else {
			require.Equal(t, "sha256:089e70ad64b18b82c0a30a7cdbe20fbda1ba2406f04ec97faedbe4a6cc5f46e9", d.String())
		}
	})
}

type testLogger struct{}

func (l testLogger) Logf(string, ...interface{}) {}
