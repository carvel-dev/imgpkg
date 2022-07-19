// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"fmt"
	"os"
	"testing"

	goui "github.com/cppforlife/go-cli-ui/ui"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/image"
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
				ui := goui.NewConfUI(goui.NewNoopLogger())
				img, err := image.NewFileImage(tInfo.path, nil)
				require.NoError(t, err)
				folder, err := os.MkdirTemp("", "test")
				require.NoError(t, err)

				imgDir := image.NewDirImage(folder, img, ui)
				require.NoError(t, imgDir.AsDirectory())

				// TODO: maybe check if the files are present
			})
		}
	}
}
