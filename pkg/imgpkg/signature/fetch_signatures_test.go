// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package signature_test

import (
	"fmt"
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/imageset"
	"carvel.dev/imgpkg/pkg/imgpkg/lockconfig"
	"carvel.dev/imgpkg/pkg/imgpkg/signature"
	"carvel.dev/imgpkg/pkg/imgpkg/signature/signaturefakes"
	regname "github.com/google/go-containerregistry/pkg/name"
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

	t.Run("denied errors, when calling Fetch, work as not found", func(t *testing.T) {
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
			return imageset.UnprocessedImageRef{}, signature.AccessDeniedErr{}
		})

		args := imageset.NewUnprocessedImageRefs()
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img1@sha256:6716afd7a68262a37d3f67681ed9dedf3b882938ad777f268f44d68894531f7a"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:a40a266ca606d8dcbac60b4bb1ec42128ba7063f5eed3a997ec4546edc6cf209"})
		signatures, err := subject.Fetch(args)
		require.NoError(t, err)

		require.Equal(t, 2, signatures.Length())
	})

	t.Run("denied errors are provided as part of the error, when calling FetchForImageRefs", func(t *testing.T) {
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
			return imageset.UnprocessedImageRef{}, signature.AccessDeniedErr{}
		})

		var args []lockconfig.ImageRef
		args = append(args, lockconfig.ImageRef{Image: "registry.io/img@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0"})
		args = append(args, lockconfig.ImageRef{Image: "registry.io/img1@sha256:6716afd7a68262a37d3f67681ed9dedf3b882938ad777f268f44d68894531f7a"})
		args = append(args, lockconfig.ImageRef{Image: "registry.io/img2@sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b"})
		args = append(args, lockconfig.ImageRef{Image: "registry.io/img3@sha256:a40a266ca606d8dcbac60b4bb1ec42128ba7063f5eed3a997ec4546edc6cf209"})
		signatures, err := subject.FetchForImageRefs(args)
		require.Error(t, err)
		errs, ok := err.(*signature.FetchError)
		require.True(t, ok, "Unexpected error found %+v, while expecting a FetchError", err)
		require.Len(t, errs.AllErrors, 2)
		require.EqualError(t, errs.AllErrors[0], "access denied")
		require.EqualError(t, errs.AllErrors[1], "access denied")

		require.Len(t, signatures, 2)
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
