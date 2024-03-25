// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/imagetar"
	"carvel.dev/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Assertion struct {
	T                    *testing.T
	logger               *Logger
	signatureKeyLocation string
}

func (a *Assertion) ImagesDigestIsOnTar(tarFilePath string, imagesDigestRef ...string) {
	a.T.Helper()
	imagesOrIndexes, err := imagetar.NewTarReader(tarFilePath).Read()
	require.NoError(a.T, err)

	for _, imageOrIndex := range imagesOrIndexes {
		imageRefFromTar := imageOrIndex.Ref()
		found := false
		for _, ref := range imagesDigestRef {
			if imageRefFromTar == ref {
				found = true
			}
		}
		require.Truef(a.T, found, "unexpected image ref (%s) referenced in manifest.json", imageRefFromTar)
	}
}

func (a *Assertion) AssertBundleLock(path, expectedBundleRef, expectedTag string) {
	a.T.Helper()
	bLock, err := lockconfig.NewBundleLockFromPath(path)
	require.NoError(a.T, err)

	assert.Equal(a.T, expectedBundleRef, bLock.Bundle.Image)
	assert.Equal(a.T, expectedTag, bLock.Bundle.Tag)

	// Do not replace bundleLockKind or bundleLockAPIVersion with consts
	// BundleLockKind or BundleLockAPIVersion.
	// This is done to prevent updating the const.
	assert.Equal(a.T, "imgpkg.carvel.dev/v1alpha1", bLock.APIVersion)
	assert.Equal(a.T, "BundleLock", bLock.Kind)
}

func (a *Assertion) AssertImagesLock(path string, images []lockconfig.ImageRef) {
	a.T.Helper()
	imagesLock, err := lockconfig.NewImagesLockFromPath(path)
	require.NoError(a.T, err)
	require.Len(a.T, imagesLock.Images, len(images))

	for i, image := range images {
		got := imagesLock.Images[i]
		assert.Equalf(a.T, image.Image, got.Image, "image %d", i)
		assert.Equalf(a.T, image.Annotations, got.Annotations, "image %d", i)
	}

	// Do not replace imagesLockKind or imagesLockAPIVersion with consts
	// ImagesLockKind or ImagesLockAPIVersion.
	// This is done to prevent updating the const.
	assert.Equal(a.T, "ImagesLock", imagesLock.Kind)
	assert.Equal(a.T, "imgpkg.carvel.dev/v1alpha1", imagesLock.APIVersion)
}

func (a *Assertion) ValidateImagesPresenceInRegistry(refs []string) error {
	a.T.Helper()
	for _, refString := range refs {
		ref, err := name.ParseReference(refString)
		require.NoError(a.T, err)
		if _, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
			return fmt.Errorf("validating image %s: %v", refString, err)
		}
	}
	return nil
}

func (a *Assertion) ValidateCosignSignature(refs []string) {
	for _, ref := range refs {
		cmdArgs := []string{"verify", "-key", filepath.Join(a.signatureKeyLocation, "cosign.pub"), ref}
		a.logger.Debugf("Running 'cosign %s'\n", strings.Join(cmdArgs, " "))

		cmd := exec.Command("cosign", cmdArgs...)
		var stderr, stdout bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &stdout

		err := cmd.Run()
		assert.NoError(a.T, err, fmt.Sprintf("error: %s", stderr.String()))
	}
}
