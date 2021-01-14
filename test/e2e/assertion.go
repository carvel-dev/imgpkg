// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

type assertion struct {
	t *testing.T
}

func (a *assertion) isBundleLockFile(path, bundleImgRef string) {
	a.t.Helper()
	bundleBs, err := ioutil.ReadFile(path)
	if err != nil {
		a.t.Fatalf("Could not read bundle lock file: %s", err)
	}

	// Keys are written in alphabetical order
	expectedYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
bundle:
  image: %s@sha256:[A-Fa-f0-9]{64}
  tag: latest
kind: BundleLock
`, bundleImgRef)

	if !regexp.MustCompile(expectedYml).Match(bundleBs) {
		a.t.Fatalf("Regex did not match; diff expected...actual:\n%v\n", diffText(expectedYml, string(bundleBs)))
	}
}

func (a *assertion) imagesDigestIsOnTar(tarFilePath string, imagesDigestRef ...string) {
	a.t.Helper()
	imagesOrIndexes, err := imagetar.NewTarReader(tarFilePath).Read()
	if err != nil {
		a.t.Fatalf("failed to read tar: %v", err)
	}

	for _, imageOrIndex := range imagesOrIndexes {
		imageRefFromTar := imageOrIndex.Ref()
		found := false
		for _, ref := range imagesDigestRef {
			if imageRefFromTar == ref {
				found = true
			}
		}

		if !found {
			a.t.Fatalf("unexpected image ref (%s) referenced in manifest.json", imageRefFromTar)
		}
	}
}

func (a *assertion) assertBundleLock(path, expectedBundleRef, expectedTag string) {
	a.t.Helper()
	bLock, err := lockconfig.NewBundleLockFromPath(path)
	if err != nil {
		a.t.Fatalf("unable to read bundle lock file: %s", err)
	}

	if bLock.Bundle.Image != expectedBundleRef {
		a.t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'",
			expectedBundleRef, bLock.Bundle.Image)
	}

	if bLock.Bundle.Tag != expectedTag {
		a.t.Fatalf("expected lock output to contain tag '%s', got '%s'",
			expectedBundleRef, bLock.Bundle.Tag)
	}

	// Do not replace bundleLockKind or bundleLockAPIVersion with consts
	// BundleLockKind or BundleLockAPIVersion.
	// This is done to prevent updating the const.
	bundleLockKind := "BundleLock"
	bundleLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if bLock.APIVersion != bundleLockAPIVersion {
		a.t.Fatalf("expected apiVersion to equal: %s, but got: %s",
			bundleLockAPIVersion, bLock.APIVersion)
	}

	if bLock.Kind != bundleLockKind {
		a.t.Fatalf("expected Kind to equal: %s, but got: %s", bundleLockKind, bLock.Kind)
	}
}

func (a *assertion) assertImagesLock(path string, images []lockconfig.ImageRef) {
	a.t.Helper()
	imagesLock, err := lockconfig.NewImagesLockFromPath(path)
	if err != nil {
		a.t.Fatalf("unable to read bundle lock file: %s", err)
	}

	if len(images) != len(imagesLock.Images) {
		a.t.Fatalf("expected number of images is different from the received\nExpected:\n %d\n Got:%d\n", len(images), len(imagesLock.Images))
	}

	var errors []string
	for i, image := range images {
		got := imagesLock.Images[i]
		if image.Image != got.Image {
			errors = append(errors, fmt.Sprintf("Image %d: expected image URL to be '%s' but got '%s'", i, image.Image, got.Image))
		}
		if !reflect.DeepEqual(image.Annotations, got.Annotations) {
			errors = append(errors, fmt.Sprintf("Image %d: expected image annotations to be '%v' but got '%v'", i, image.Annotations, got.Annotations))
		}
	}

	if len(errors) > 0 {
		a.t.Fatalf("Images in the lock file do not match expected: %s", strings.Join(errors, "\n\t"))
	}

	// Do not replace imagesLockKind or imagesLockAPIVersion with consts
	// ImagesLockKind or ImagesLockAPIVersion.
	// This is done to prevent updating the const.
	imagesLockKind := "ImagesLock"
	imagesLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if imagesLock.APIVersion != imagesLockAPIVersion {
		a.t.Fatalf("expected apiVersion to equal: %s, but got: %s",
			imagesLockAPIVersion, imagesLock.APIVersion)
	}

	if imagesLock.Kind != imagesLockKind {
		a.t.Fatalf("expected Kind to equal: %s, but got: %s", imagesLockKind, imagesLock.Kind)
	}
}

func (a *assertion) validateImagesPresenceInRegistry(refs []string) error {
	a.t.Helper()
	for _, refString := range refs {
		ref, _ := name.ParseReference(refString)
		if _, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
			return fmt.Errorf("validating image %s: %v", refString, err)
		}
	}
	return nil
}
