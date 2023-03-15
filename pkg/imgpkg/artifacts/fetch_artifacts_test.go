// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package artifacts_test

import (
	"fmt"
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/artifacts"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/artifacts/artifactsfakes"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
)

func TestArtifactsRetriever_Signatures(t *testing.T) {
	t.Run("it does not add signatures that cannot be found", func(t *testing.T) {
		fakeArtifactFinder := &artifactsfakes.FakeFinder{}
		subject := artifacts.NewArtifacts(fakeArtifactFinder, 2)
		fakeArtifactFinder.SignatureCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			availableResults := map[string]imageset.UnprocessedImageRef{
				"sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0": {DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"},
				"sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b": {DigestRef: "registry.io/img2@sha256:be154cc2b1211a9f98f4d708f4266650c9129784d0485d4507d9b0fa05d928b6", Tag: "some-other-tag"},
			}
			if res, ok := availableResults[digest.DigestStr()]; ok {
				return res, nil
			}
			return imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{}
		})
		fakeArtifactFinder.SBOMReturns(imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{})
		fakeArtifactFinder.AttestationReturns(imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{})

		args := imageset.NewUnprocessedImageRefs()
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img1@sha256:6716afd7a68262a37d3f67681ed9dedf3b882938ad777f268f44d68894531f7a"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b"})
		artifact, err := subject.Fetch(args)
		require.NoError(t, err)

		require.Equal(t, 3, fakeArtifactFinder.SBOMCallCount(), "Checks SBOM existence for 3 images")
		require.Equal(t, 3, fakeArtifactFinder.AttestationCallCount(), "Checks Attestations existence for 3 images")
		require.Equal(t, 3, fakeArtifactFinder.SignatureCallCount(), "Checks signatures for all images, since no SBOMs or Attestations are present")

		require.Len(t, artifact.All(), 2)
		sign1 := artifact.All()[0]
		assert.Equal(t, imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:be154cc2b1211a9f98f4d708f4266650c9129784d0485d4507d9b0fa05d928b6", Tag: "some-other-tag"}, sign1)
		sign2 := artifact.All()[1]
		assert.Equal(t, imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"}, sign2)
	})

	t.Run("it does not add SBOM that cannot be found", func(t *testing.T) {
		fakeArtifactFinder := &artifactsfakes.FakeFinder{}
		subject := artifacts.NewArtifacts(fakeArtifactFinder, 2)
		fakeArtifactFinder.SignatureReturns(imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{})
		fakeArtifactFinder.SBOMCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			availableResults := map[string]imageset.UnprocessedImageRef{
				"sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0": {DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"},
				"sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b": {DigestRef: "registry.io/img2@sha256:be154cc2b1211a9f98f4d708f4266650c9129784d0485d4507d9b0fa05d928b6", Tag: "some-other-tag"},
			}
			if res, ok := availableResults[digest.DigestStr()]; ok {
				return res, nil
			}
			return imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{}
		})
		fakeArtifactFinder.AttestationReturns(imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{})

		args := imageset.NewUnprocessedImageRefs()
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img1@sha256:6716afd7a68262a37d3f67681ed9dedf3b882938ad777f268f44d68894531f7a"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b"})
		artifact, err := subject.Fetch(args)
		require.NoError(t, err)

		require.Equal(t, 3, fakeArtifactFinder.SBOMCallCount(), "Checks SBOM existence for 3 images")
		require.Equal(t, 3, fakeArtifactFinder.AttestationCallCount(), "Checks Attestations existence for 3 images")
		require.Equal(t, 5, fakeArtifactFinder.SignatureCallCount(), "Checks signatures for all images and SBOMs")

		require.Len(t, artifact.All(), 2)
		sbom1 := artifact.All()[0]
		assert.Equal(t, imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:be154cc2b1211a9f98f4d708f4266650c9129784d0485d4507d9b0fa05d928b6", Tag: "some-other-tag"}, sbom1)
		sbom2 := artifact.All()[1]
		assert.Equal(t, imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"}, sbom2)
	})

	t.Run("it does not add Attestations that cannot be found", func(t *testing.T) {
		fakeArtifactFinder := &artifactsfakes.FakeFinder{}
		subject := artifacts.NewArtifacts(fakeArtifactFinder, 2)

		fakeArtifactFinder.SignatureReturns(imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{})
		fakeArtifactFinder.SBOMReturns(imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{})
		fakeArtifactFinder.AttestationCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			availableResults := map[string]imageset.UnprocessedImageRef{
				"sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0": {DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"},
				"sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b": {DigestRef: "registry.io/img2@sha256:be154cc2b1211a9f98f4d708f4266650c9129784d0485d4507d9b0fa05d928b6", Tag: "some-other-tag"},
			}
			if res, ok := availableResults[digest.DigestStr()]; ok {
				return res, nil
			}
			return imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{}
		})

		args := imageset.NewUnprocessedImageRefs()
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img1@sha256:6716afd7a68262a37d3f67681ed9dedf3b882938ad777f268f44d68894531f7a"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b"})
		artifact, err := subject.Fetch(args)
		require.NoError(t, err)

		require.Equal(t, 3, fakeArtifactFinder.SBOMCallCount(), "Checks SBOM existence for 3 images")
		require.Equal(t, 3, fakeArtifactFinder.AttestationCallCount(), "Checks Attestations existence for 3 images")
		require.Equal(t, 5, fakeArtifactFinder.SignatureCallCount(), "Checks signatures for all images and Attestations")

		require.Len(t, artifact.All(), 2)
		att1 := artifact.All()[0]
		assert.Equal(t, imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:be154cc2b1211a9f98f4d708f4266650c9129784d0485d4507d9b0fa05d928b6", Tag: "some-other-tag"}, att1)
		att2 := artifact.All()[1]
		assert.Equal(t, imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"}, att2)
	})

	t.Run("denied errors are provided as part of the error", func(t *testing.T) {
		fakeArtifactFinder := &artifactsfakes.FakeFinder{}
		subject := artifacts.NewArtifacts(fakeArtifactFinder, 2)
		fakeArtifactFinder.SignatureCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			availableResults := map[string]imageset.UnprocessedImageRef{
				"sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0": {DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"},
				"sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b": {DigestRef: "registry.io/img2@sha256:be154cc2b1211a9f98f4d708f4266650c9129784d0485d4507d9b0fa05d928b6", Tag: "some-other-tag"},
			}
			if res, ok := availableResults[digest.DigestStr()]; ok {
				return res, nil
			}
			return imageset.UnprocessedImageRef{}, artifacts.AccessDeniedErr{}
		})
		fakeArtifactFinder.SBOMReturns(imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{})
		fakeArtifactFinder.AttestationReturns(imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{})

		var args []lockconfig.ImageRef
		args = append(args, lockconfig.ImageRef{Image: "registry.io/img@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0"})
		args = append(args, lockconfig.ImageRef{Image: "registry.io/img1@sha256:6716afd7a68262a37d3f67681ed9dedf3b882938ad777f268f44d68894531f7a"})
		args = append(args, lockconfig.ImageRef{Image: "registry.io/img2@sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b"})
		args = append(args, lockconfig.ImageRef{Image: "registry.io/img3@sha256:a40a266ca606d8dcbac60b4bb1ec42128ba7063f5eed3a997ec4546edc6cf209"})
		signatures, err := subject.FetchForImageRefs(args)
		require.Error(t, err)
		errs, ok := err.(*artifacts.FetchError)
		require.True(t, ok, "Unexpected error found '%+v', while expecting a FetchError", err)
		require.Len(t, errs.AllErrors, 2)
		require.EqualError(t, errs.AllErrors[0], "access denied")
		require.EqualError(t, errs.AllErrors[1], "access denied")

		require.Len(t, signatures, 2)
	})

	t.Run("it retrieve signatures for sboms and attestations", func(t *testing.T) {
		fakeArtifactFinder := &artifactsfakes.FakeFinder{}
		subject := artifacts.NewArtifacts(fakeArtifactFinder, 2)

		fakeArtifactFinder.SignatureCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			if digest.DigestStr() == "sha256:26c68657ccce2cb0a31b330cb0be2b5e108d467f641c62e13ab40cbec258c68d" {
				return imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:859ab6768a6f26a79bc42b231664111317d095a4f04e4b6fe79ce37b3d199097", Tag: "sbom-sign"}, nil
			} else if digest.DigestStr() == "sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93" {
				return imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:d2b53584f580310186df7a2055ce3ff83cc0df6caacf1e3489bff8cf5d0af5d8", Tag: "att-sign"}, nil
			}
			return imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{}
		})
		fakeArtifactFinder.SBOMCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			if digest.DigestStr() == "sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0" {
				return imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:26c68657ccce2cb0a31b330cb0be2b5e108d467f641c62e13ab40cbec258c68d"}, nil
			}
			return imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{}
		})
		fakeArtifactFinder.AttestationCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			availableResults := map[string]imageset.UnprocessedImageRef{
				"sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0": {DigestRef: "registry.io/img@sha256:cf31af331f38d1d7158470e095b132acd126a7180a54f263d386da88eb681d93", Tag: "some-tag"},
				"sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b": {DigestRef: "registry.io/img2@sha256:be154cc2b1211a9f98f4d708f4266650c9129784d0485d4507d9b0fa05d928b6", Tag: "some-other-tag"},
			}
			if res, ok := availableResults[digest.DigestStr()]; ok {
				return res, nil
			}
			return imageset.UnprocessedImageRef{}, artifacts.NotFoundErr{}
		})

		args := imageset.NewUnprocessedImageRefs()
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img1@sha256:6716afd7a68262a37d3f67681ed9dedf3b882938ad777f268f44d68894531f7a"})
		args.Add(imageset.UnprocessedImageRef{DigestRef: "registry.io/img2@sha256:56cb33b3b4bc45509c5ff7513ddc6ed78764f9ad5165cc32826e04da49d5462b"})
		artifact, err := subject.Fetch(args)
		require.NoError(t, err)

		require.Equal(t, 3, fakeArtifactFinder.SBOMCallCount(), "Checks SBOM existence for 3 images")
		require.Equal(t, 3, fakeArtifactFinder.AttestationCallCount(), "Checks Attestations existence for 3 images")
		require.Equal(t, 6, fakeArtifactFinder.SignatureCallCount(), "Checks signatures for all images, SBOM and Attestations")

		require.Len(t, artifact.All(), 5)
		assert.Contains(t, artifact.All(), imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:859ab6768a6f26a79bc42b231664111317d095a4f04e4b6fe79ce37b3d199097", Tag: "sbom-sign"})
		assert.Contains(t, artifact.All(), imageset.UnprocessedImageRef{DigestRef: "registry.io/img@sha256:d2b53584f580310186df7a2055ce3ff83cc0df6caacf1e3489bff8cf5d0af5d8", Tag: "att-sign"})
	})

	t.Run("it returns error when returned error is not sign.NotFound", func(t *testing.T) {
		fakeArtifactFinder := &artifactsfakes.FakeFinder{}
		subject := artifacts.NewArtifacts(fakeArtifactFinder, 2)
		fakeArtifactFinder.SignatureCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			if fakeArtifactFinder.SignatureCallCount() == 1 {
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

	t.Run("it returns error when returned error is not sbom.NotFound", func(t *testing.T) {
		fakeArtifactFinder := &artifactsfakes.FakeFinder{}
		subject := artifacts.NewArtifacts(fakeArtifactFinder, 2)
		fakeArtifactFinder.SBOMCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			if fakeArtifactFinder.SignatureCallCount() == 1 {
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

	t.Run("it returns error when returned error is not sbom.NotFound", func(t *testing.T) {
		fakeArtifactFinder := &artifactsfakes.FakeFinder{}
		subject := artifacts.NewArtifacts(fakeArtifactFinder, 2)
		fakeArtifactFinder.AttestationCalls(func(digest regname.Digest) (imageset.UnprocessedImageRef, error) {
			if fakeArtifactFinder.SignatureCallCount() == 1 {
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
