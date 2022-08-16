// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/image"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
)

func TestDirImage(t *testing.T) {
	tests := struct {
		info map[string][]struct {
			path string
			key  string
		}
	}{
		info: map[string][]struct {
			path string
			key  string
		}{
			"macos": {
				{
					key:  "new",
					path: "test_assets/img_tar.macos.new.tar",
				},
				{
					key:  "old",
					path: "test_assets/img_tar.macos.old.tar",
				},
			},
			"windows": {
				{
					key:  "new",
					path: "test_assets/img_tar.windows.new.tar",
				},
				{
					key:  "old",
					path: "test_assets/img_tar.windows.old.tar",
				},
			},
		},
	}
	for osName, test := range tests.info {
		for _, tInfo := range test {
			t.Run(fmt.Sprintf("open image tar created in %s from %s imgpkg version", osName, tInfo.key), func(t *testing.T) {
				img, err := image.NewFileImage(tInfo.path, nil)
				require.NoError(t, err)
				folder, err := os.MkdirTemp("", "test")
				require.NoError(t, err)

				imgDir := image.NewDirImage(folder, img, util.NewNoopLevelLogger())
				require.NoError(t, imgDir.AsDirectory())

				// TODO: maybe check if the files are present
			})
		}
	}
}
