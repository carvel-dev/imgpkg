// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

func TestPullErrors(t *testing.T) {
	t.Run("fails when no image, bundle or lockfile are provided", func(t *testing.T) {
		pull := PullOptions{OutputPath: "/tmp/some/place"}
		err := pull.Run()
		require.Error(t, err)
		require.ErrorContains(t, err, "Expected either image or bundle reference")
	})

	t.Run("fails when more than one source is provided", func(t *testing.T) {
		pull := PullOptions{OutputPath: "/tmp/some/place", ImageFlags: ImageFlags{"image@123456"}, BundleFlags: BundleFlags{"my-bundle"}, LockInputFlags: LockInputFlags{LockFilePath: "lockpath"}}
		err := pull.Run()
		require.Error(t, err)
		require.ErrorContains(t, err, "Expected only one of image, bundle, or lock")
	})

	t.Run("fails when recursive flag is provided but not the bundle flag", func(t *testing.T) {
		pull := PullOptions{OutputPath: "/tmp/some/place", ImageFlags: ImageFlags{"image@123456"}, BundleRecursiveFlags: BundleRecursiveFlags{Recursive: true}}
		err := pull.Run()
		require.Error(t, err)
		require.ErrorContains(t, err, "Cannot use --recursive (-r) flag when pulling a bundle")
	})

	t.Run("fails when arguments are provided without a flag", func(t *testing.T) {
		confUI := ui.NewConfUI(ui.NewNoopLogger())
		defer confUI.Flush()

		imgpkgCmd := NewDefaultImgpkgCmd(confUI)
		imgpkgCmd.SetArgs([]string{"pull", "k8slt/image", "-o", "/tmp"})
		err := imgpkgCmd.Execute()
		require.Error(t, err)
		require.ErrorContains(t, err, "command 'imgpkg pull' does not accept extra arguments 'k8slt/image'")
	})

	t.Run("fails when pull bundle using -i flag", func(t *testing.T) {
		confUI := ui.NewConfUI(ui.NewNoopLogger())
		defer confUI.Flush()

		bundleName := "some/bundle"
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle").
			WithEveryImageFromPath("test_assets/image_with_config", map[string]string{})

		defer fakeRegistry.CleanUp()

		fakeRegistry.Build()
		pull := PullOptions{
			ImageFlags:         ImageFlags{fakeRegistry.ReferenceOnTestServer(bundleName)},
			OutputPath:         "/tmp/some/place",
			ImageIsBundleCheck: true, // This is the default value
			ui:                 confUI,
		}
		err := pull.Run()
		require.Error(t, err)
		require.ErrorContains(t, err, "Expected bundle flag when pulling a bundle (hint: Use -b instead of -i for bundles)")
	})

	t.Run("fails when pull image using -b flag", func(t *testing.T) {
		confUI := ui.NewConfUI(ui.NewNoopLogger())
		defer confUI.Flush()

		imgName := "some/image"
		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		fakeRegistry.WithRandomImage(imgName)

		defer fakeRegistry.CleanUp()

		fakeRegistry.Build()
		pull := PullOptions{
			BundleFlags:        BundleFlags{Bundle: fakeRegistry.ReferenceOnTestServer(imgName)},
			OutputPath:         "/tmp/some/place",
			ImageIsBundleCheck: true, // This is the default value
			ui:                 confUI,
		}
		err := pull.Run()
		require.Error(t, err)
		require.ErrorContains(t, err, "Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
	})
}
