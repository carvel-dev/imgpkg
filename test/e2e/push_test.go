// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushBundleInImageLockErr(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	bundleDir := env.BundleFactory.CreateBundleDir(bundleYAML, imagesYAML)
	out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(t, out))
	bundleDigestRef := env.Image + bundleDigest

	imagesLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, bundleDigestRef)
	env.BundleFactory.AddFileToBundle(filepath.Join(".imgpkg", "images.yml"), imagesLockYAML)

	var stderrBs bytes.Buffer
	_, err := imgpkg.RunWithOpts([]string{"push", "-b", env.Image, "-f", bundleDir},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()
	if err == nil {
		t.Fatalf("Expected pushing to fail because of bundle ref in image lock file, but got success")
	}
	if !strings.Contains(errOut, "Expected image lock to not contain bundle reference") {
		t.Fatalf("Expected pushing to fail because of bundle ref in image lock file got: %s", errOut)
	}
}

func TestPushBundleOfBundles(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	bundleDigestRef := ""
	bundleDir := env.BundleFactory.CreateBundleDir(bundleYAML, imagesYAML)
	logger.Section("create inner bundle", func() {
		out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
		bundleDigestRef = fmt.Sprintf("%s@%s", env.Image, extractDigest(t, out))
	})

	logger.Section("create new bundle with bundles", func() {
		imagesLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, bundleDigestRef)
		env.BundleFactory.AddFileToBundle(filepath.Join(".imgpkg", "images.yml"), imagesLockYAML)

		imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir, "--experimental-recursive-bundle"})
	})
}
