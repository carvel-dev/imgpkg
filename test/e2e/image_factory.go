// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

type imageFactory struct {
	assets *assets
	t *testing.T
}

func (i *imageFactory) pushSimpleAppImageWithRandomFile(imgpkg Imgpkg, imgRef string) string {
	i.t.Helper()
	imgDir := i.assets.createAndCopy("simple-image")
	// Add file to ensure we have a different digest
	i.assets.addFileToFolder(filepath.Join(imgDir, "random-file.txt"), randStringRunes(500))

	out := imgpkg.Run([]string{"push", "--tty", "-i", imgRef, "-f", imgDir})
	return fmt.Sprintf("@%s", extractDigest(i.t, out))
}

func (i imageFactory) assertImagesLock(path string, images []lockconfig.ImageRef) error {
	imagesLock, err := lockconfig.NewImagesLockFromPath(path)
	if err != nil {
		return fmt.Errorf("unable to read bundle lock file: %s", err)
	}

	if len(images) != len(imagesLock.Images) {
		return fmt.Errorf("expected number of images is different from the received\nExpected:\n %d\n Got:%d\n", len(images), len(imagesLock.Images))
	}
	var errors []string
	for i, image := range images {
		got := imagesLock.Images[i]
		if image.Image != got.Image {
			errors = append(errors, fmt.Sprintf("Image %d: expected image URL to be '%s' but got '%s'",i, image.Image, got.Image))
		}
		if !reflect.DeepEqual(image.Annotations, got.Annotations) {
			errors = append(errors, fmt.Sprintf("Image %d: expected image annotations to be '%v' but got '%v'", i, image.Annotations, got.Annotations))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("Images in the lock file do not match expected: %s", strings.Join(errors, "\n\t"))
	}

	// Do not replace imagesLockKind or imagesLockAPIVersion with consts
	// ImagesLockKind or ImagesLockAPIVersion.
	// This is done to prevent updating the const.
	imagesLockKind := "ImagesLock"
	imagesLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if imagesLock.APIVersion != imagesLockAPIVersion {
		return fmt.Errorf("expected apiVersion to equal: %s, but got: %s",
			imagesLockAPIVersion, imagesLock.APIVersion)
	}

	if imagesLock.Kind != imagesLockKind {
		return fmt.Errorf("expected Kind to equal: %s, but got: %s", imagesLockKind, imagesLock.Kind)
	}
	return nil
}
