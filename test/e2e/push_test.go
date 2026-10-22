// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"carvel.dev/imgpkg/test/helpers"
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

func TestPushWithJSONFlagPrintsErrorToStderr(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	missingDir := filepath.Join(env.Assets.CreateTempFolder("push-json-error"), "missing")

	var stderr bytes.Buffer
	out, err := imgpkg.RunWithOpts([]string{"push", "-b", env.Image, "-f", missingDir, "--json"},
		helpers.RunOpts{AllowError: true, StderrWriter: &stderr})

	require.EqualError(t, err, "Execution error: stdout: '' stderr: '' error: 'exit status 1'")
	require.Empty(t, out)
	require.Equal(t, fmt.Sprintf("imgpkg: Error: lstat %s: no such file or directory\n", missingDir), stderr.String())
}
