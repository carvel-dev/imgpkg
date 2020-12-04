// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package lockfiles

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestImageLockNonDigestUnmarshalError(t *testing.T) {
	imageLockYaml := []byte(`apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: nginx:v1`)

	var imageLock ImageLock
	err := yaml.Unmarshal(imageLockYaml, &imageLock)

	if err == nil {
		t.Fatalf("Expected unmarshal to error")
	}

	if msg := err.Error(); !(strings.Contains(msg, "to be in digest form") && strings.Contains(msg, "nginx:v1")) {
		t.Fatalf("Expected unmarshal to fail due to tag ref in lock file")
	}
}

func TestReadImageLockFileSucceed(t *testing.T) {
	imgLock, err := ReadImageLockFile("testdata/imagelock.yml")
	if err != nil {
		t.Fatalf("Expected no error but received one: %v", err)
	}

	if imgLock.Kind != ImagesLockKind {
		t.Fatalf("Expected %s but go %s", ImagesLockKind, imgLock.Kind)
	}

	if len(imgLock.Spec.Images) != 1 {
		t.Fatalf("Expected only 1 image but got %d", len(imgLock.Spec.Images))
	}

	expected := "index.docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0"
	if imgLock.Spec.Images[0].Image != expected {
		t.Fatalf("Expected image to be %s but got %s", expected, imgLock.Spec.Images[0].Image)
	}
}

func TestReadImageLockFileFail(t *testing.T) {
	_, err := ReadImageLockFile("path/does/not/exist")
	if err == nil {
		t.Fatalf("Expected error from non existent path but received none")
	}
}

func TestReadBundleLockFileSucceed(t *testing.T) {
	bundleLock, err := ReadBundleLockFile("testdata/bundlelock.yml")
	if err != nil {
		t.Fatalf("Expected no error but received one: %v", err)
	}

	if bundleLock.Kind != BundleLockKind {
		t.Fatalf("Expected %s but go %s", ImagesLockKind, bundleLock.Kind)
	}

	expectedURL := "docker.io/my-app@sha256:b12026c7a0a6a1756a82a2a74ac759e9a7036523faca0e33dbddebc214e097df\n"
	if bundleLock.Spec.Image.DigestRef == expectedURL {
		t.Fatalf("Expected Bundle ref to be %s but got %s", expectedURL, bundleLock.Spec.Image.DigestRef)
	}

	expectedTag := "1.0"
	if bundleLock.Spec.Image.OriginalTag == expectedTag {
		t.Fatalf("Expected Bundle tag to be %s but got %s", expectedTag, bundleLock.Spec.Image.OriginalTag)
	}
}

func TestReadBundleLockFileFail(t *testing.T) {
	_, err := ReadBundleLockFile("path/does/not/exist")
	if err == nil {
		t.Fatalf("Expected error from non existent path but received none")
	}
}
