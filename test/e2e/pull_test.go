// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

func TestPullImageLockRewrite(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	pushDir := filepath.Join(os.TempDir(), "imgpkg-test-imagelock-rewrite-push")
	pullDir := filepath.Join(os.TempDir(), "imgpkg-test-imagelock-rewrite-pull")
	cleanUp := func() { os.RemoveAll(pushDir); os.RemoveAll(pullDir) }
	defer cleanUp()

	err := os.Mkdir(pushDir, 0700)
	if err != nil {
		t.Fatalf("failed to create push directory: %v", err)
	}
	imageDigestRef := "@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6"
	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: hello-world%s
`, imageDigestRef)

	_, err = createBundleDir(pushDir, bundleYAML, imageLockYAML)
	if err != nil {
		t.Fatalf("failed to create image lock file: %v", err)
	}

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", pushDir})
	imgpkg.Run([]string{"copy", "-b", env.Image, "--to-repo", env.Image})
	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", pullDir})

	imgLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(pullDir, ".imgpkg", "images.yml"))
	if err != nil {
		t.Fatalf("could not read images lock: %v", err)
	}

	actualImageRef := imgLock.Images[0].Image
	expectedImageRef := env.Image + imageDigestRef
	if actualImageRef != expectedImageRef {
		t.Fatalf("Expected images lock to be updated with bundle repository; diff expected...actual:\n%v\n", diffText(expectedImageRef, actualImageRef))
	}
}
