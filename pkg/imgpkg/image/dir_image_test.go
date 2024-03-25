// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/image"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("When extracting tar that contains files and folders with permissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("permissions values are different in windows because of umask")
		}
		img, err := image.NewFileImage(filepath.Join("test_assets", "img_tar_with_permissions.tar"), nil)
		require.NoError(t, err)
		folder := t.TempDir()

		imgDir := image.NewDirImage(folder, img, util.NewNoopLogger())
		require.NoError(t, imgDir.AsDirectory())

		// this is true using umask 022 that will remove the write bit from the permissions
		expectedPermissions := map[string]string{
			"folder_all":                      "drwxr-xr-x",
			"folder_all/exec_perm_all.sh":     "-rwxr-xr-x",
			"folder_all/some_file.txt":        "-rw-r--r--",
			"folder_group":                    "drwxr-xr-x",
			"folder_group/exec_perm_group.sh": "-rwxr-xr-x",
			"folder_group/some_other.txt":     "-rw-r--r--",
		}
		filepath.WalkDir(folder, func(path string, d fs.DirEntry, _ error) error {
			if path == folder {
				return nil
			}
			key := strings.ReplaceAll(path, folder+string(filepath.Separator), "")
			key = strings.ReplaceAll(key, "/", string(filepath.Separator))
			fInfo, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, expectedPermissions[key], fInfo.Mode().String(), fmt.Sprintf("validating file %s", key))
			return nil
		})
	})
}
