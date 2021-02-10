// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset/imagesetfakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

type FakeRegistry struct {
	state map[string]*ImageOrImageIndexWithTarPath
	t     *testing.T
}

func NewFakeRegistry(t *testing.T) *FakeRegistry {
	return &FakeRegistry{state: map[string]*ImageOrImageIndexWithTarPath{}, t: t}
}

func (r *FakeRegistry) Build() *imagesetfakes.FakeImagesReaderWriter {
	fakeRegistry := &imagesetfakes.FakeImagesReaderWriter{}
	fakeRegistry.GenericCalls(func(reference name.Reference) (descriptor v1.Descriptor, err error) {
		mediaType := types.OCIManifestSchema1
		if val, found := r.state[reference.Name()]; found {
			if val.image != nil {
				mediaType, err = val.image.MediaType()
				digest, err := val.image.Digest()
				if err != nil {
					r.t.Fatal(err.Error())
				}
				return v1.Descriptor{
					MediaType: mediaType,
					Digest:    digest,
				}, nil
			}

			imageIndex := val.imageIndex
			digest, err := imageIndex.Digest()
			if err != nil {
				r.t.Fatal(err.Error())
			}
			mediaType, err = imageIndex.MediaType()
			return v1.Descriptor{
				MediaType: mediaType,
				Digest:    digest,
			}, nil
		}

		return v1.Descriptor{}, fmt.Errorf("FakeRegistry: GenericCall: image [%s] not found", reference.Name())
	})

	fakeRegistry.WriteImageStub = func(reference name.Reference, v v1.Image) error {
		r.state[reference.Name()] = &ImageOrImageIndexWithTarPath{fakeRegistry: r, t: r.t, image: v, imageName: reference.Name()}
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
	if err != nil {
		r.t.Fatalf("Failed trying to compress %s: %s", path, err)
	}
	label := map[string]string{"dev.carvel.imgpkg.bundle": ""}

	bundle, err := image.NewFileImage(tarballLayer.Name(), label)
	if err != nil {
		r.t.Fatalf("unable to create image from file: %s", err)
	}

	r.updateState(bundleName, bundle, nil, path)
	return BundleInfo{r, path}

}

func (r *FakeRegistry) updateState(imageName string, image v1.Image, imageIndex v1.ImageIndex, path string) {
	imgName, err := name.ParseReference(imageName)
	if err != nil {
		r.t.Fatalf("unable to parse reference: %s", err)
	}

	imageOrImageIndexWithTarPath := &ImageOrImageIndexWithTarPath{fakeRegistry: r, t: r.t, imageName: imageName, image: image, imageIndex: imageIndex, path: path}
	r.state[imgName.Name()] = imageOrImageIndexWithTarPath

	if image != nil {
		digest, err := image.Digest()
		if err != nil {
			r.t.Fatalf("unable to parse reference: %s", err)
		}
		r.state[imgName.Context().Name()+"@"+digest.String()] = imageOrImageIndexWithTarPath
	}
}

func (r *FakeRegistry) WithImageFromPath(imageNameFromTest string, path string) *ImageOrImageIndexWithTarPath {
	tarballLayer, err := compress(path)
	if err != nil {
		r.t.Fatalf("Failed trying to compress %s: %s", path, err)
	}

	fileImage, err := image.NewFileImage(tarballLayer.Name(), nil)
	if err != nil {
		r.t.Fatalf("Failed trying to build a file image%s", err)
	}

	r.updateState(imageNameFromTest, fileImage, nil, path)
	reference, err := name.ParseReference(imageNameFromTest)
	if err != nil {
		r.t.Fatalf("Failed trying to get image name: %s", err)
	}

	return r.state[reference.Name()]
}

func (r *FakeRegistry) WithARandomImageIndex(imageName string) {
	index, err := random.Index(1024, 1, 1)
	if err != nil {
		r.t.Fatal(err.Error())
	}
	manifest, err := index.IndexManifest()
	if err != nil {
		r.t.Fatal(err.Error())
	}

	imageUsedInIndex, err := index.Image(manifest.Manifests[0].Digest)
	if err != nil {
		r.t.Fatal(err.Error())
	}

	r.updateState(imageName, imageUsedInIndex, index, "")
}

func (r *FakeRegistry) WithNonDistributableLayerInImage(imageNames ...string) {
	for _, imageName := range imageNames {
		reference, err := name.ParseReference(imageName)
		if err != nil {
			r.t.Fatalf("Failed trying to parse an image name%s", err)
		}

		layer, err := random.Layer(1024, types.OCIUncompressedRestrictedLayer)
		if err != nil {
			r.t.Fatalf("unable to create a layer %s", err)
		}

		imageWithARestrictedLayer, err := mutate.AppendLayers(r.state[reference.Name()].image, layer)
		if err != nil {
			r.t.Fatalf("unable to append a layer %s", err)
		}

		r.updateState(imageName, imageWithARestrictedLayer, r.state[reference.Name()].imageIndex, r.state[reference.Name()].path)
	}
}

func (r *FakeRegistry) CleanUp() {
	for _, tarPath := range r.state {
		os.Remove(tarPath.path)
	}
}

type BundleInfo struct {
	r          *FakeRegistry
	BundlePath string
}

func (b BundleInfo) WithEveryImageFrom(path string) *FakeRegistry {
	imgLockPath := filepath.Join(b.BundlePath, ".imgpkg", "images.yml")
	imgLock, err := lockconfig.NewImagesLockFromPath(imgLockPath)
	if err != nil {
		b.r.t.Fatalf("Got error: %s", err.Error())
	}

	for _, img := range imgLock.Images {
		b.r.WithImageFromPath(img.Image, path)
	}
	return b.r
}

type ImageOrImageIndexWithTarPath struct {
	fakeRegistry *FakeRegistry
	imageName    string
	image        v1.Image
	imageIndex   v1.ImageIndex
	path         string
	t            *testing.T
}

func (r *ImageOrImageIndexWithTarPath) WithNonDistributableLayer() {
	layer, err := random.Layer(1024, types.OCIUncompressedRestrictedLayer)
	if err != nil {
		r.t.Fatalf("unable to create a layer %s", err)
	}
	r.image, err = mutate.AppendLayers(r.image, layer)
	if err != nil {
		r.t.Fatalf("unable to append a layer %s", err)
	}

	reference, err := name.ParseReference(r.imageName)
	if err != nil {
		r.t.Fatalf("unable to parse reference: %s", err)
	}

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
