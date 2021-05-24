// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package signature_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/signature"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCosign_Signature(t *testing.T) {
	t.Run("it returns the signature when it can be found", func(t *testing.T) {
		logger := &helpers.Logger{}
		regBuilder := helpers.NewFakeRegistry(t, logger)
		sigImg := regBuilder.WithRandomImage("some-image")
		signatureTag := fmt.Sprintf("sha256-%s.sig", strings.Split(sigImg.Digest, ":")[1])
		sigImg.Tag = signatureTag
		reg := regBuilder.Build()
		defer regBuilder.CleanUp()

		subject := signature.NewCosign(reg)
		imgDigest, err := name.NewDigest(sigImg.RefDigest)
		require.NoError(t, err)
		signature, err := subject.Signature(imgDigest)
		require.NoError(t, err)
		assert.Equal(t, sigImg.RefDigest, signature.DigestRef)
		assert.Equal(t, signatureTag, signature.Tag)
	})

	t.Run("it returns sign.NotFound when image with the signature tag cannot be found", func(t *testing.T) {
		logger := &helpers.Logger{}
		regBuilder := helpers.NewFakeRegistry(t, logger)
		sigImg := regBuilder.WithRandomImage("some-image")
		reg := regBuilder.Build()
		defer regBuilder.CleanUp()

		subject := signature.NewCosign(reg)
		imgDigest, err := name.NewDigest(sigImg.RefDigest)
		require.NoError(t, err)

		_, err = subject.Signature(imgDigest)
		require.Error(t, err)

		_, ok := err.(signature.NotFound)
		require.True(t, ok)
	})
}
