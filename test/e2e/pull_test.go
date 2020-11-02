// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/k14s/imgpkg/pkg/imgpkg/cmd"
	"gopkg.in/yaml.v2"
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
	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: hello-world%s
`, imageDigestRef)

	_, err = createBundleDir(pushDir, bundleYAML, imgsYml)
	if err != nil {
		t.Fatalf("failed to create image lock file: %v", err)
	}

	imgpkg.Run([]string{"push", "-b", env.Image, "-f", pushDir})
	imgpkg.Run([]string{"copy", "-b", env.Image, "--to-repo", env.Image})
	imgpkg.Run([]string{"pull", "-b", env.Image, "-o", pullDir})

	iLockBytes, err := ioutil.ReadFile(filepath.Join(pullDir, cmd.BundleDir, imageFile))
	if err != nil {
		t.Fatalf("could not read images lock: %v", err)
	}
	var iLock cmd.ImageLock
	err = yaml.Unmarshal(iLockBytes, &iLock)
	if err != nil {
		t.Fatalf("could not unmarshal images lock: %v", err)
	}

	actualImageRef := iLock.Spec.Images[0].Image
	expectedImageRef := env.Image + imageDigestRef
	if actualImageRef != expectedImageRef {
		t.Fatalf("Expected images lock to be updated with bundle repository: %s, but got: %s", expectedImageRef, actualImageRef)
	}
}
