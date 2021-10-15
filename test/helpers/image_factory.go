// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	goui "github.com/cppforlife/go-cli-ui/ui"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/require"
	ctlimg "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/image"
)

type ImageFactory struct {
	Assets               *Assets
	T                    *testing.T
	signatureKeyLocation string
	logger               *Logger
}

func (i *ImageFactory) ImageDigest(imgRef string) string {
	imageRef, err := name.ParseReference(imgRef, name.WeakValidation)
	require.NoError(i.T, err)
	img, err := remote.Image(imageRef, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(i.T, err)

	digest, err := img.Digest()
	require.NoError(i.T, err)
	return digest.String()
}

func (i *ImageFactory) PushImageWithANonDistributableLayer(imgRef string, mediaType types.MediaType) string {
	imageRef, err := name.ParseReference(imgRef, name.WeakValidation)
	require.NoError(i.T, err)

	layer, err := random.Layer(1024, mediaType)
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

func (i *ImageFactory) PushSimpleAppImageWithRandomFileWithAuth(imgpkg Imgpkg, imgRef string, host, username, password string) string {
	i.T.Helper()
	imgDir := i.Assets.CreateAndCopySimpleApp("simple-image")
	// Add file to ensure we have a different digest
	i.Assets.AddFileToFolder(filepath.Join(imgDir, "random-file.txt"), randString(500))

	out, err := imgpkg.RunWithOpts([]string{"push", "--tty", "-i", imgRef, "-f", imgDir}, RunOpts{
		EnvVars: []string{"IMGPKG_REGISTRY_HOSTNAME=" + host, "IMGPKG_REGISTRY_USERNAME=" + username, "IMGPKG_REGISTRY_PASSWORD=" + password},
	})
	require.NoError(i.T, err)
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

func (i *ImageFactory) SignImage(imgRef string) string {
	cmdArgs := []string{"sign", "-key", filepath.Join(i.signatureKeyLocation, "cosign.key"), imgRef}
	i.logger.Debugf("Running 'cosign %s'\n", strings.Join(cmdArgs, " "))

	cmd := exec.Command("cosign", cmdArgs...)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "COSIGN_PASSWORD=")

	var stderr, stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err := cmd.Run()
	require.NoError(i.T, err, fmt.Sprintf("error: %s", stderr.String()))

	stderrStr := stderr.String()
	match := regexp.MustCompile(":(sha256-[0123456789abcdef]{64}.*)").FindStringSubmatch(stderrStr)
	require.Len(i.T, match, 2)
	return match[1]
}

func (i *ImageFactory) Download(imgRef, location string) {
	imageReg, err := name.ParseReference(imgRef, name.WeakValidation)
	require.NoError(i.T, err)
	img, err := remote.Image(imageReg, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	require.NoError(i.T, err)

	output := bytes.NewBufferString("")
	writerUI := goui.NewWriterUI(output, output, nil)
	err = ctlimg.NewDirImage(filepath.Join(location), img, writerUI).AsDirectory()
	require.NoError(i.T, err)
}
