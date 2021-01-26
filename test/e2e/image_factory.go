// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"path/filepath"
	"testing"
)

type imageFactory struct {
	assets *assets
	t      *testing.T
}

func (i *imageFactory) PushImageWithANonDistributableLayer(imgRef string) string {
	imageRef, err := name.ParseReference(imgRef, name.WeakValidation)

	image, err := random.Image(1024, 1)
	if err != nil {
		i.t.Fatalf(err.Error())
	}
	layer, err := random.Layer(1024, types.OCIUncompressedRestrictedLayer)
	if err != nil {
		i.t.Fatalf(err.Error())
	}
	image, err = mutate.Append(empty.Image, mutate.Addendum{
		Layer: layer,
		URLs:  []string{"http://"},
	})
	if err != nil {
		i.t.Fatalf(err.Error())
	}

	err = remote.WriteLayer(imageRef.Context(), layer, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		i.t.Fatalf(err.Error())
	}
	err = remote.Write(imageRef, image, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		i.t.Fatalf(err.Error())
	}

	digest, err := layer.Digest()
	if err != nil {
		i.t.Fatalf(err.Error())
	}
	return digest.String()
}

func (i *imageFactory) PushSimpleAppImageWithRandomFile(imgpkg Imgpkg, imgRef string) string {
	i.t.Helper()
	imgDir := i.assets.CreateAndCopySimpleApp("simple-image")
	// Add file to ensure we have a different digest
	i.assets.AddFileToFolder(filepath.Join(imgDir, "random-file.txt"), randString(500))

	out := imgpkg.Run([]string{"push", "--tty", "-i", imgRef, "-f", imgDir})
	return fmt.Sprintf("@%s", extractDigest(i.t, out))
}
