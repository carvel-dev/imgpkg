// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset/imagesetfakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/stretchr/testify/require"
)

type FakeRegistry struct {
	state map[string]*ImageOrImageIndexWithTarPath
	t     *testing.T
}

func NewFakeRegistry(t *testing.T) *FakeRegistry {
	return &FakeRegistry{state: map[string]*ImageOrImageIndexWithTarPath{}, t: t}
}

type ImageOrImageIndexWithTarPath struct {
	fakeRegistry *FakeRegistry
	imageName    string
	image        v1.Image
	imageIndex   v1.ImageIndex
	path         string
	t            *testing.T
	RefDigest    string
}

type BundleInfo struct {
	r          *FakeRegistry
	BundlePath string
	RefDigest  string
}

func (b BundleInfo) WithEveryImageFrom(path string) *FakeRegistry {
	imgLockPath := filepath.Join(b.BundlePath, ".imgpkg", "images.yml")
	imgLock, err := lockconfig.NewImagesLockFromPath(imgLockPath)
	require.NoError(b.r.t, err)

	for _, img := range imgLock.Images {
		b.r.WithImageFromPath(img.Image, path)
	}
	return b.r
}

func (r *FakeRegistry) Build() *imagesetfakes.FakeImagesReaderWriter {
	fakeRegistry := &imagesetfakes.FakeImagesReaderWriter{}
	getDescriptor := func(reference name.Reference, r *FakeRegistry) (v1.Descriptor, error) {
		mediaType := types.OCIManifestSchema1
		if val, found := r.state[reference.Name()]; found {
			if val.image != nil {
				mediaType, err := val.image.MediaType()
				digest, err := val.image.Digest()
				require.NoError(r.t, err)

				return v1.Descriptor{
					MediaType: mediaType,
					Digest:    digest,
				}, nil
			}

			imageIndex := val.imageIndex
			digest, err := imageIndex.Digest()
			require.NoError(r.t, err)

			mediaType, err = imageIndex.MediaType()
			return v1.Descriptor{
				MediaType: mediaType,
				Digest:    digest,
			}, nil
		}

		return v1.Descriptor{}, fmt.Errorf("FakeRegistry: GenericCall: image [%s] not found", reference.Name())
	}

	fakeRegistry.GenericCalls(func(reference name.Reference) (descriptor v1.Descriptor, err error) {
		return getDescriptor(reference, r)
	})

	fakeRegistry.DigestCalls(func(reference name.Reference) (v1.Hash, error) {
		if val, found := r.state[reference.Name()]; found {
			if val.image != nil {
				return val.image.Digest()
			}
			return val.imageIndex.Digest()
		}

		return v1.Hash{}, fmt.Errorf("FakeRegistry: DigestCall: image [%s] not found", reference.Name())
	})

	fakeRegistry.GetCalls(func(reference name.Reference) (*regremote.Descriptor, error) {
		if val, found := r.state[reference.Name()]; found {
			descriptor, err := getDescriptor(reference, r)
			if err != nil {
				r.t.Fatal(err.Error())
			}
			if val.image != nil {
				manifest, err := val.image.RawManifest()
				if err != nil {
					r.t.Fatal(err.Error())
				}
				return &regremote.Descriptor{
					Descriptor: descriptor,
					Manifest:   manifest,
				}, nil
			}

			manifest, err := val.imageIndex.RawManifest()
			if err != nil {
				r.t.Fatal(err.Error())
			}
			return &regremote.Descriptor{
				Descriptor: descriptor,
				Manifest:   manifest,
			}, nil
		}

		return &regremote.Descriptor{}, fmt.Errorf("FakeRegistry: GetCall: image [%s] not found", reference.Name())
	})

	fakeRegistry.WriteImageStub = func(reference name.Reference, v v1.Image) error {
		r.state[reference.Name()] = &ImageOrImageIndexWithTarPath{fakeRegistry: r, t: r.t, image: v, imageName: reference.Name()}
		return nil
	}

	fakeRegistry.MultiWriteStub = func(reference map[name.Reference]regremote.Taggable, _ int) error {
		for ref, taggable := range reference {
			r.state[ref.Name()] = &ImageOrImageIndexWithTarPath{fakeRegistry: r, t: r.t, image: taggable.(v1.Image), imageName: ref.Name()}
		}
		return nil
	}

	fakeRegistry.ImageStub = func(reference name.Reference) (v v1.Image, err error) {
		if bundle, found := r.state[reference.Name()]; found {
			return bundle.image, nil
		}
		return nil, fmt.Errorf("Did not find bundle in fake registry: %s", reference.Context().Name())
	}

	fakeRegistry.IndexStub = func(reference name.Reference) (v1.ImageIndex, error) {
		if imageIndexFromState, found := r.state[reference.Name()]; found {
			return imageIndexFromState.imageIndex, nil
		}
		return nil, fmt.Errorf("Did not find image index in fake registry: %s", reference.Context().Name())
	}

	return fakeRegistry
}

func (r *FakeRegistry) WithBundleFromPath(bundleName string, path string) BundleInfo {
	tarballLayer, err := compress(path)
	require.NoError(r.t, err, "compressing the bundle")
	label := map[string]string{"dev.carvel.imgpkg.bundle": ""}

	bundle, err := image.NewFileImage(tarballLayer.Name(), label)
	require.NoError(r.t, err, "create image from tar")

	b := r.updateState(bundleName, bundle, nil, path)
	return BundleInfo{r, path, b.RefDigest}
}

func (r *FakeRegistry) WithRandomBundle(bundleName string) BundleInfo {
	bundle, err := random.Image(500, 5)
	bundle, err = mutate.ConfigFile(bundle, &v1.ConfigFile{
		Config: v1.Config{
			Labels: map[string]string{"dev.carvel.imgpkg.bundle": "true"},
		},
	})
	require.NoError(r.t, err, "create image from tar")

	b := r.updateState(bundleName, bundle, nil, "")
	return BundleInfo{r, "", b.RefDigest}
}

func (r *FakeRegistry) WithImageFromPath(imageNameFromTest string, path string) *ImageOrImageIndexWithTarPath {
	tarballLayer, err := compress(path)
	require.NoError(r.t, err, "compressing the path")

	fileImage, err := image.NewFileImage(tarballLayer.Name(), nil)
	require.NoError(r.t, err, "create image from tar")

	return r.updateState(imageNameFromTest, fileImage, nil, path)
}

func (r *FakeRegistry) WithRandomImage(imageNameFromTest string) *ImageOrImageIndexWithTarPath {
	img, err := random.Image(500, 3)
	require.NoError(r.t, err, "create image from tar")

	return r.updateState(imageNameFromTest, img, nil, "")
}

func (r *FakeRegistry) CopyImage(img ImageOrImageIndexWithTarPath, to string) *ImageOrImageIndexWithTarPath {
	newImg := img

	digest, err := newImg.image.Digest()
	require.NoError(r.t, err)
	newImg.RefDigest = to + "@" + digest.String()
	r.state[newImg.RefDigest] = &newImg
	return &newImg
}

func (r *FakeRegistry) CopyBundleImage(bundleInfo BundleInfo, to string) BundleInfo {
	digest := strings.Split(bundleInfo.RefDigest, "@")[1]
	newBundle := *r.state[bundleInfo.RefDigest]
	newBundle.RefDigest = to + "@" + digest
	r.state[newBundle.RefDigest] = &newBundle
	return BundleInfo{r, "", newBundle.RefDigest}
}

func (r *FakeRegistry) WithARandomImageIndex(imageName string) {
	index, err := random.Index(1024, 1, 1)
	require.NoError(r.t, err)

	manifest, err := index.IndexManifest()
	require.NoError(r.t, err)

	imageUsedInIndex, err := index.Image(manifest.Manifests[0].Digest)
	require.NoError(r.t, err)

	r.updateState(imageName, imageUsedInIndex, index, "")
}

func (r *FakeRegistry) WithNonDistributableLayerInImage(imageNames ...string) {
	for _, imageName := range imageNames {
		reference, err := name.ParseReference(imageName)
		require.NoErrorf(r.t, err, "parse reference: %s", imageName)

		layer, err := random.Layer(1024, types.OCIUncompressedRestrictedLayer)
		require.NoErrorf(r.t, err, "create layer: %s", imageName)

		imageWithARestrictedLayer, err := mutate.AppendLayers(r.state[reference.Name()].image, layer)
		require.NoErrorf(r.t, err, "add layer: %s", imageName)

		r.updateState(imageName, imageWithARestrictedLayer, r.state[reference.Name()].imageIndex, r.state[reference.Name()].path)
	}
}

func (r *FakeRegistry) updateState(imageName string, image v1.Image, imageIndex v1.ImageIndex, path string) *ImageOrImageIndexWithTarPath {
	imgName, err := name.ParseReference(imageName)
	require.NoError(r.t, err)

	imageOrImageIndexWithTarPath := &ImageOrImageIndexWithTarPath{fakeRegistry: r, t: r.t, imageName: imageName, image: image, imageIndex: imageIndex, path: path}
	r.state[imgName.Name()] = imageOrImageIndexWithTarPath

	if image != nil {
		digest, err := image.Digest()
		require.NoError(r.t, err)

		imageOrImageIndexWithTarPath.RefDigest = imgName.Context().Name() + "@" + digest.String()
		r.state[imageOrImageIndexWithTarPath.RefDigest] = imageOrImageIndexWithTarPath
	}
	return imageOrImageIndexWithTarPath
}

func (r *FakeRegistry) CleanUp() {
	for _, tarPath := range r.state {
		if tarPath.path != "" {
			os.Remove(tarPath.path)
		}
	}
}

func (r *ImageOrImageIndexWithTarPath) WithNonDistributableLayer() {
	layer, err := random.Layer(1024, types.OCIUncompressedRestrictedLayer)
	require.NoError(r.t, err)

	r.image, err = mutate.AppendLayers(r.image, layer)
	require.NoError(r.t, err)

	reference, err := name.ParseReference(r.imageName)
	require.NoError(r.t, err)

	r.fakeRegistry.updateState(reference.Name(), r.image, r.imageIndex, r.path)
}

func compress(src string) (*os.File, error) {
	_, err := os.Stat(src)
	if err != nil {
		return nil, fmt.Errorf("Unable to compress because file not found: %s", err)
	}
	tempTarFile, err := ioutil.TempFile(os.TempDir(), "compressed-layer")
	if err != nil {
		return nil, err
	}
	tw := tar.NewWriter(tempTarFile)

	// walk through every file in the folder
	filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, file)
		if err != nil {
			return err
		}

		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})

	// produce tar
	if err := tw.Close(); err != nil {
		return tempTarFile, err
	}

	return tempTarFile, err
}
