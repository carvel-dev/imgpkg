// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

func TestPullImageLockRewrite(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	imageDigestRef := "@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6"
	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: hello-world%s
`, imageDigestRef)

	bundleDir := env.BundleFactory.CreateBundleDir(bundleYAML, imageLockYAML)

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", bundleDir})
	imgpkg.Run([]string{"copy", "-b", env.Image, "--to-repo", env.Image})

	pullDir := env.Assets.CreateTempFolder("pull-rewrite-lock")
	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", pullDir})

	expectedImageRef := env.Image + imageDigestRef
	env.Assert.AssertImagesLock(filepath.Join(pullDir, ".imgpkg", "images.yml"), []lockconfig.ImageRef{{Image: expectedImageRef}})
}
