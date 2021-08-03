// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package signature_test

import (
	"fmt"
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/signature"
	"github.com/k14s/imgpkg/pkg/imgpkg/signature/signaturefakes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignatureRetriever_Signatures(t *testing.T) {
	t.Run("it does not add signatures that cannot be found", func(t *testing.T) {
		fakeSignatureFinder := &signaturefakes.FakeFinder{}
		subject := signature.NewSignatures(fakeSignatureFinder, 2)
		fakeSignatureFinder.SignatureCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			availableResults := map[string]imageset.UnprocessedImageRef{
				"sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0": {DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"},
				"sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b": {DigestRef: "registry.io/img2@sha256:be154cc2b1211a9f98f4d708f4266650c9129784d0485d4507d9b0fa05d928b6", Tag: "some-other-tag"},
			}
			if res, ok := availableResults[digest.DigestStr()]; ok {
				return res, nil
			}
			return imageset.UnprocessedImageRef{}, signature.NotFoundErr{}
		})

		args := imageset.NewUnprocessedImageRefs()
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img1@sha256:6716afd7a68262a37d3f67681ed9dedf3b882938ad777f268f44d68894531f7a"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b"})
		signatures, err := subject.Fetch(args)
		require.NoError(t, err)

		require.Len(t, signatures.All(), 2)
		sign1 := signatures.All()[0]
		assert.Equal(t, imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:be154cc2b1211a9f98f4d708f4266650c9129784d0485d4507d9b0fa05d928b6", Tag: "some-other-tag"}, sign1)
		sign2 := signatures.All()[1]
		assert.Equal(t, imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"}, sign2)
	})

	t.Run("it returns error when returned error is not sign.NotFound", func(t *testing.T) {
		fakeSignatureFinder := &signaturefakes.FakeFinder{}
		subject := signature.NewSignatures(fakeSignatureFinder, 2)
		fakeSignatureFinder.SignatureCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			if fakeSignatureFinder.SignatureCallCount() == 1 {
				return imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"}, nil
			}
			return imageset.UnprocessedImageRef{}, fmt.Errorf("should make function fail")
		})

		args := imageset.NewUnprocessedImageRefs()
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:6716afd7a68262a37d3f67681ed9dedf3b882938ad777f268f44d68894531f7a"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img1@sha256:6716afd7a68262a37d3f67681ed9dedf3b882938ad777f268f44d68894531f7a"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b"})
		_, err := subject.Fetch(args)
		require.Error(t, err)
	})
}
