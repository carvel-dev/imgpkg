// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/require"
)

type ImageFactory struct {
	Assets *Assets
	T      *testing.T
}

func (i *ImageFactory) PushImageWithANonDistributableLayer(imgRef string) string {
	imageRef, err := name.ParseReference(imgRef, name.WeakValidation)
	require.NoError(i.T, err)

	layer, err := random.Layer(1024, types.OCIUncompressedRestrictedLayer)
	require.NoError(i.T, err)
	digest, err := layer.Digest()
	require.NoError(i.T, err)
	image, err := mutate.Append(empty.Image, mutate.Addendum{
		Layer: layer,
		URLs:  []string{fmt.Sprintf("%s://%s/v2/%s/blobs/%s", imageRef.Context().Registry.Scheme(), imageRef.Context().RegistryStr(), imageRef.Context().RepositoryStr(), digest)},
	})
	require.NoError(i.T, err)

	err = remote.WriteLayer(imageRef.Context(), layer, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(i.T, err)
	err = remote.Write(imageRef, image, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(i.T, err)

	return digest.String()
}

func (i *ImageFactory) PushSimpleAppImageWithRandomFile(imgpkg Imgpkg, imgRef string) string {
	i.T.Helper()
	imgDir := i.Assets.CreateAndCopySimpleApp("simple-image")
	// Add file to ensure we have a different digest
	i.Assets.AddFileToFolder(filepath.Join(imgDir, "random-file.txt"), randString(500))

	out := imgpkg.Run([]string{"push", "--tty", "-i", imgRef, "-f", imgDir})
	return fmt.Sprintf("@%s", ExtractDigest(i.T, out))
}

func (i *ImageFactory) PushImageWithLayerSize(imgRef string, size int64) string {
	imageRef, err := name.ParseReference(imgRef, name.WeakValidation)
	require.NoError(i.T, err)

	layer, err := random.Layer(size, types.OCIUncompressedLayer)
	require.NoError(i.T, err)
	digest, err := layer.Digest()
	require.NoError(i.T, err)
	image, err := mutate.Append(empty.Image, mutate.Addendum{
		Layer: layer,
		URLs:  []string{fmt.Sprintf("%s://%s/v2/%s/blobs/%s", imageRef.Context().Registry.Scheme(), imageRef.Context().RegistryStr(), imageRef.Context().RepositoryStr(), digest)},
	})
	require.NoError(i.T, err)

	err = remote.WriteLayer(imageRef.Context(), layer, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(i.T, err)
	err = remote.Write(imageRef, image, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(i.T, err)

	return digest.String()
}

func (i *ImageFactory) PushImageIndex(imgRef string) {
	imageRef, err := name.ParseReference(imgRef, name.WeakValidation)
	require.NoError(i.T, err)

	index, err := random.Index(1024, 1, 2)
	require.NoError(i.T, err)

	err = remote.WriteIndex(imageRef, index, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(i.T, err)
}
