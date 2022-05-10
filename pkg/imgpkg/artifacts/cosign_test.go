// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package artifacts_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/artifacts"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
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

		subject := artifacts.NewCosign(reg)
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

		subject := artifacts.NewCosign(reg)
		imgDigest, err := name.NewDigest(sigImg.RefDigest)
		require.NoError(t, err)

		_, err = subject.Signature(imgDigest)
		require.Error(t, err)

		_, ok := err.(artifacts.NotFoundErr)
		require.True(t, ok)
	})
}
func TestCosign_Attestation(t *testing.T) {
	t.Run("it returns the attestation when it can be found", func(t *testing.T) {
		logger := &helpers.Logger{}
		regBuilder := helpers.NewFakeRegistry(t, logger)
		sigImg := regBuilder.WithRandomImage("some-image")
		signatureTag := fmt.Sprintf("sha256-%s.att", strings.Split(sigImg.Digest, ":")[1])
		sigImg.Tag = signatureTag
		reg := regBuilder.Build()
		defer regBuilder.CleanUp()

		subject := artifacts.NewCosign(reg)
		imgDigest, err := name.NewDigest(sigImg.RefDigest)
		require.NoError(t, err)
		attn, err := subject.Attestation(imgDigest)
		require.NoError(t, err)
		assert.Equal(t, sigImg.RefDigest, attn.DigestRef)
		assert.Equal(t, signatureTag, attn.Tag)
	})

	t.Run("it returns artifacts.NotFound when image with the attestation tag cannot be found", func(t *testing.T) {
		logger := &helpers.Logger{}
		regBuilder := helpers.NewFakeRegistry(t, logger)
		sigImg := regBuilder.WithRandomImage("some-image")
		reg := regBuilder.Build()
		defer regBuilder.CleanUp()

		subject := artifacts.NewCosign(reg)
		imgDigest, err := name.NewDigest(sigImg.RefDigest)
		require.NoError(t, err)

		_, err = subject.Attestation(imgDigest)
		require.Error(t, err)

		_, ok := err.(artifacts.NotFoundErr)
		require.True(t, ok)
	})
}

func TestCosign_SBOM(t *testing.T) {
	t.Run("it returns the SBOM when it can be found", func(t *testing.T) {
		logger := &helpers.Logger{}
		regBuilder := helpers.NewFakeRegistry(t, logger)
		sigImg := regBuilder.WithRandomImage("some-image")
		signatureTag := fmt.Sprintf("sha256-%s.sbom", strings.Split(sigImg.Digest, ":")[1])
		sigImg.Tag = signatureTag
		reg := regBuilder.Build()
		defer regBuilder.CleanUp()

		subject := artifacts.NewCosign(reg)
		imgDigest, err := name.NewDigest(sigImg.RefDigest)
		require.NoError(t, err)
		sbom, err := subject.SBOM(imgDigest)
		require.NoError(t, err)
		assert.Equal(t, sigImg.RefDigest, sbom.DigestRef)
		assert.Equal(t, signatureTag, sbom.Tag)
	})

	t.Run("it returns artifacts.NotFound when image with the SBOM tag cannot be found", func(t *testing.T) {
		logger := &helpers.Logger{}
		regBuilder := helpers.NewFakeRegistry(t, logger)
		sigImg := regBuilder.WithRandomImage("some-image")
		reg := regBuilder.Build()
		defer regBuilder.CleanUp()

		subject := artifacts.NewCosign(reg)
		imgDigest, err := name.NewDigest(sigImg.RefDigest)
		require.NoError(t, err)

		_, err = subject.SBOM(imgDigest)
		require.Error(t, err)

		_, ok := err.(artifacts.NotFoundErr)
		require.True(t, ok)
	})
}
