// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
	"golang.org/x/sys/unix"
)

func TestPull(t *testing.T) {
	logger := &helpers.Logger{}

	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: *logger, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	t.Run("Image - copies the User Permission to group and other", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping test as this is a known issue: https://github.com/vmware-tanzu/carvel-imgpkg/issues/270")
		}

		folder := env.Assets.CreateTempFolder("simple-image")
		env.Assets.AddFileToFolderWithPermissions(filepath.Join(folder, "all-on-user-only"), "some text", 0755)
		env.Assets.AddFileToFolderWithPermissions(filepath.Join(folder, "read-on-user-only"), "some text", 0455)
		env.Assets.AddFileToFolderWithPermissions(filepath.Join(folder, "read-write-on-user-only"), "some text", 0655)

		out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", folder})
		imgDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))

		pullDir := filepath.Join(env.Assets.CreateTempFolder("pull-dir-simple-image"), "pull-dir")
		imageRef := fmt.Sprintf("%s%s", env.Image, imgDigest)

		oldMask := unix.Umask(0)
		defer unix.Umask(oldMask)

		imgpkg.Run([]string{"pull", "-i", imageRef, "-o", pullDir})

		logger.Section("ensures that pull folder is created with 0777 permissions", func() {
			info, err := os.Stat(pullDir)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0777).String(), (info.Mode() & 0777).String(), "outer folder permission should be set to 0777")
		})

		info, err := os.Stat(filepath.Join(pullDir, "all-on-user-only"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0700).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
		assert.Equal(t, os.FileMode(0070).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
		assert.Equal(t, os.FileMode(0007).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
		info, err = os.Stat(filepath.Join(pullDir, "read-on-user-only"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0400).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
		assert.Equal(t, os.FileMode(0040).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
		assert.Equal(t, os.FileMode(0004).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
		info, err = os.Stat(filepath.Join(pullDir, "read-write-on-user-only"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
		assert.Equal(t, os.FileMode(0060).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
		assert.Equal(t, os.FileMode(0006).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
	})

	t.Run("Image - copies the User Permission to group and other but skips execution because umask is set to 0111", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping test as this is a known issue: https://github.com/vmware-tanzu/carvel-imgpkg/issues/270")
		}

		folder := env.Assets.CreateTempFolder("simple-image")
		innerFolder := filepath.Join(folder, "some-folder")
		env.Assets.AddFolder(innerFolder, 0755)
		env.Assets.AddFileToFolderWithPermissions(filepath.Join(innerFolder, "read-on-user-only"), "some text", 0455)
		env.Assets.AddFileToFolderWithPermissions(filepath.Join(folder, "all-on-user-only"), "some text", 0755)
		env.Assets.AddFileToFolderWithPermissions(filepath.Join(folder, "read-on-user-only"), "some text", 0455)
		env.Assets.AddFileToFolderWithPermissions(filepath.Join(folder, "read-write-on-user-only"), "some text", 0655)

		out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", folder})
		imgDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))

		pullDir := env.Assets.CreateTempFolder("pull-dir-simple-image")
		imageRef := fmt.Sprintf("%s%s", env.Image, imgDigest)

		oldMask := unix.Umask(0011)
		defer unix.Umask(oldMask)

		imgpkg.Run([]string{"pull", "-i", imageRef, "-o", pullDir})
		logger.Section("ensures that pull folder is created with 0766 permissions", func() {
			info, err := os.Stat(pullDir)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0766).String(), (info.Mode() & 0777).String(), "outer folder permission should be set to 0766")
		})

		logger.Section("check permissions inside a subfolder", func() {
			info, err := os.Stat(filepath.Join(pullDir, "some-folder"))
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0700).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
			assert.Equal(t, os.FileMode(0060).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
			assert.Equal(t, os.FileMode(0006).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
			info, err = os.Stat(filepath.Join(pullDir, "some-folder", "read-on-user-only"))
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0400).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
			assert.Equal(t, os.FileMode(0040).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
			assert.Equal(t, os.FileMode(0004).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
		})

		info, err := os.Stat(filepath.Join(pullDir, "all-on-user-only"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0700).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
		assert.Equal(t, os.FileMode(0060).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
		assert.Equal(t, os.FileMode(0006).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
		info, err = os.Stat(filepath.Join(pullDir, "read-on-user-only"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0400).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
		assert.Equal(t, os.FileMode(0040).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
		assert.Equal(t, os.FileMode(0004).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
		info, err = os.Stat(filepath.Join(pullDir, "read-write-on-user-only"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
		assert.Equal(t, os.FileMode(0060).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
		assert.Equal(t, os.FileMode(0006).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
	})

	t.Run("Bundle - copies the User Permission to group and other", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping test as this is a known issue: https://github.com/vmware-tanzu/carvel-imgpkg/issues/270")
		}

		bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, helpers.ImagesYAML)
		env.Assets.AddFileToFolderWithPermissions(filepath.Join(bundleDir, "all-on-user-only"), "some text", 0755)
		env.Assets.AddFileToFolderWithPermissions(filepath.Join(bundleDir, "read-on-user-only"), "some text", 0455)
		env.Assets.AddFileToFolderWithPermissions(filepath.Join(bundleDir, "read-write-on-user-only"), "some text", 0655)

		out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
		imgDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))

		pullDir := env.Assets.CreateTempFolder("pull-dir-simple-image")
		imageRef := fmt.Sprintf("%s%s", env.Image, imgDigest)

		oldMask := unix.Umask(0)
		defer unix.Umask(oldMask)

		imgpkg.Run([]string{"pull", "-b", imageRef, "-o", pullDir})

		info, err := os.Stat(filepath.Join(pullDir, "all-on-user-only"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0700).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
		assert.Equal(t, os.FileMode(0070).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
		assert.Equal(t, os.FileMode(0007).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
		info, err = os.Stat(filepath.Join(pullDir, "read-on-user-only"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0400).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
		assert.Equal(t, os.FileMode(0040).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
		assert.Equal(t, os.FileMode(0004).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
		info, err = os.Stat(filepath.Join(pullDir, "read-write-on-user-only"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600).String(), (info.Mode() & 0700).String(), "user permission doesnt match")
		assert.Equal(t, os.FileMode(0060).String(), (info.Mode() & 0070).String(), "group permission doesnt match")
		assert.Equal(t, os.FileMode(0006).String(), (info.Mode() & 0007).String(), "other permission doesnt match")
	})
}
