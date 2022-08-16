// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/image"
)

func TestTarImage(t *testing.T) {
	logger := testLogger{}
	t.Run("Ensure image tar as the same SHA", func(t *testing.T) {
		tarImage := image.NewTarImage([]string{"test_assets/tar_folder"}, nil, logger)
		img, err := tarImage.AsFileImage(nil)
		require.NoError(t, err)
		d, err := img.Digest()
		require.NoError(t, err)
		if runtime.GOOS != "windows" {
			require.Equal(t, "sha256:3316053887959c59bfd01d8473f7fe20caa11c6519092fe8a3cf14d4990ec216", d.String())
		} else {
			require.Equal(t, "sha256:895251d345c46b3f9b6c2adb1443f39755187e5f314b23b78443a1bf0fa0cad2", d.String())
		}
	})
}

type testLogger struct{}

func (l testLogger) Logf(string, ...interface{}) {}
