// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushBundleOfBundles(t *testing.T) {
	env := helpers.BuildEnv(t)
	logger := helpers.Logger{}
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	bundleDigestRef := ""
	bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
	logger.Section("create inner bundle", func() {
		out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
		bundleDigestRef = fmt.Sprintf("%s@%s", env.Image, helpers.ExtractDigest(t, out))
	})

	logger.Section("create new bundle with bundles", func() {
		imagesLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, bundleDigestRef)
		env.BundleFactory.AddFileToBundle(filepath.Join(".imgpkg", "images.yml"), imagesLockYAML)

		imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir})
	})
}

func TestPushFilesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test as this is a known issue: https://github.com/vmware-tanzu/carvel-imgpkg/issues/270")
	}

	env := helpers.BuildEnv(t)
	logger := helpers.Logger{}
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	// We need this chmod, because in the github action this file permission is converted into
	// u+rw even if in the this repository the permission is correct
	require.NoError(t, os.Chmod(filepath.Join(".", "assets", "bundle_file_permissions", "read_only_config.yml"), 0400))

	logger.Section("Push bundle with different permissions files", func() {
		imgpkg.Run([]string{"push", "-f", "./assets/bundle_file_permissions", "-b", env.Image})
	})
	bundleDir := env.Assets.CreateTempFolder("bundle-location")

	logger.Section("Pull bundle", func() {
		imgpkg.Run([]string{"pull", "-b", env.Image, "-o", bundleDir})
	})

	logger.Section("Check files permissions did not change", func() {
		info, err := os.Stat(filepath.Join(bundleDir, "exec_file.sh"))
		require.NoError(t, err)
		assert.Equal(t, fs.FileMode(0700).String(), info.Mode().String(), "have -rwx------ permissions")
		info, err = os.Stat(filepath.Join(bundleDir, "read_only_config.yml"))
		require.NoError(t, err)
		assert.Equal(t, fs.FileMode(0400).String(), info.Mode().String(), "have -r-------- permissions")
		info, err = os.Stat(filepath.Join(bundleDir, "read_write_config.yml"))
		require.NoError(t, err)
		assert.Equal(t, fs.FileMode(0600).String(), info.Mode().String(), "have -rw------- permissions")
	})
}
