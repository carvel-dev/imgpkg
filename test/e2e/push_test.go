// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lf "github.com/k14s/imgpkg/pkg/imgpkg/lockfiles"
)

func TestPushBundleInImageLockErr(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	assetsPath := filepath.Join("assets", "simple-app")

	bundleDir, err := createBundleDir(assetsPath, bundleYAML, imagesYAML)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}

	out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", assetsPath})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	bundleDigestRef := env.Image + bundleDigest

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: %s
`, bundleDigestRef)
	err = ioutil.WriteFile(filepath.Join(assetsPath, lf.BundleDir, imageFile), []byte(imgsYml), 0600)
	if err != nil {
		t.Fatalf("failed to create image lock file: %v", err)
	}
	defer os.RemoveAll(bundleDir)

	var stderrBs bytes.Buffer
	_, err = imgpkg.RunWithOpts([]string{"push", "-b", env.Image, "-f", assetsPath},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()
	if err == nil {
		t.Fatalf("Expected pushing to fail because of bundle ref in image lock file, but got success")
	}
	if !strings.Contains(errOut, "Expected image lock to not contain bundle reference") {
		t.Fatalf("Expected pushing to fail because of bundle ref in image lock file got: %s", errOut)
	}
}
