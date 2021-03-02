// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

type Assertion struct {
	T *testing.T
}

func (a *Assertion) ImagesDigestIsOnTar(tarFilePath string, imagesDigestRef ...string) {
	a.T.Helper()
	imagesOrIndexes, err := imagetar.NewTarReader(tarFilePath).Read()
	if err != nil {
		a.T.Fatalf("failed to read tar: %v", err)
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
			a.T.Fatalf("unexpected image ref (%s) referenced in manifest.json", imageRefFromTar)
		}
	}
}

func (a *Assertion) AssertBundleLock(path, expectedBundleRef, expectedTag string) {
	a.T.Helper()
	bLock, err := lockconfig.NewBundleLockFromPath(path)
	if err != nil {
		a.T.Fatalf("unable to read bundle lock file: %s", err)
	}

	if bLock.Bundle.Image != expectedBundleRef {
		a.T.Fatalf("expected lock output to contain relocated ref '%s', got '%s'",
			expectedBundleRef, bLock.Bundle.Image)
	}

	if bLock.Bundle.Tag != expectedTag {
		a.T.Fatalf("expected lock output to contain tag '%s', got '%s'",
			expectedBundleRef, bLock.Bundle.Tag)
	}

	// Do not replace bundleLockKind or bundleLockAPIVersion with consts
	// BundleLockKind or BundleLockAPIVersion.
	// This is done to prevent updating the const.
	bundleLockKind := "BundleLock"
	bundleLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if bLock.APIVersion != bundleLockAPIVersion {
		a.T.Fatalf("expected apiVersion to equal: %s, but got: %s",
			bundleLockAPIVersion, bLock.APIVersion)
	}

	if bLock.Kind != bundleLockKind {
		a.T.Fatalf("expected Kind to equal: %s, but got: %s", bundleLockKind, bLock.Kind)
	}
}

func (a *Assertion) AssertImagesLock(path string, images []lockconfig.ImageRef) {
	a.T.Helper()
	imagesLock, err := lockconfig.NewImagesLockFromPath(path)
	if err != nil {
		a.T.Fatalf("unable to read bundle lock file: %s", err)
	}

	if len(images) != len(imagesLock.Images) {
		a.T.Fatalf("expected number of images is different from the received\nExpected:\n %d\n Got:%d\n", len(images), len(imagesLock.Images))
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
		a.T.Fatalf("Images in the lock file do not match expected: %s", strings.Join(errors, "\n\t"))
	}

	// Do not replace imagesLockKind or imagesLockAPIVersion with consts
	// ImagesLockKind or ImagesLockAPIVersion.
	// This is done to prevent updating the const.
	imagesLockKind := "ImagesLock"
	imagesLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if imagesLock.APIVersion != imagesLockAPIVersion {
		a.T.Fatalf("expected apiVersion to equal: %s, but got: %s",
			imagesLockAPIVersion, imagesLock.APIVersion)
	}

	if imagesLock.Kind != imagesLockKind {
		a.T.Fatalf("expected Kind to equal: %s, but got: %s", imagesLockKind, imagesLock.Kind)
	}
}

func (a *Assertion) ValidateImagesPresenceInRegistry(refs []string) error {
	a.T.Helper()
	for _, refString := range refs {
		ref, _ := name.ParseReference(refString)
		if _, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
			return fmt.Errorf("validating image %s: %v", refString, err)
		}
	}
	return nil
}
