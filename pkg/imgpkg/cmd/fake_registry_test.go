// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"archive/tar"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"strings"

	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/image/imagefakes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type FakeRegistry struct {
	state map[string]*ImageWithTarPath
	t     *testing.T
}

func NewFakeRegistry(t *testing.T) *FakeRegistry {
	return &FakeRegistry{state: map[string]*ImageWithTarPath{}, t: t}
}

func (r *FakeRegistry) Build() *imagefakes.FakeImagesReaderWriter {
	fakeRegistry := &imagefakes.FakeImagesReaderWriter{}
	fakeRegistry.GenericCalls(func(reference name.Reference) (descriptor v1.Descriptor, err error) {
		return v1.Descriptor{}, nil
	})

	fakeRegistry.ImageStub = func(reference name.Reference) (v v1.Image, err error) {
		if bundle, found := r.state[reference.Context().Name()]; found {
			return bundle.image, nil
		}
		return nil, fmt.Errorf("Did not find bundle in fake registry: %s", reference.Context().Name())
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
	r.state[bundleName] = &ImageWithTarPath{t: r.t, image: bundle, path: tarballLayer.Name()}
	return BundleInfo{r, path}

}

func (r *FakeRegistry) WithImageFromPath(name string, path string) *ImageWithTarPath {
	tarballLayer, err := compress(path)
	if err != nil {
		r.t.Fatalf("Failed trying to compress %s: %s", path, err)
	}

	image, err := image.NewFileImage(tarballLayer.Name(), nil)
	tarPath := &ImageWithTarPath{t: r.t, image: image, path: tarballLayer.Name()}
	r.state[name] = tarPath
	return tarPath
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
		imageName := strings.Split(img.Image, "@")[0]
		b.r.WithImageFromPath(imageName, path)
	}
	return b.r
}

func (r *FakeRegistry) WithNonDistributableLayerInImage(imageNames ...string) {
	for _, imageName := range imageNames {
		layer, err := random.Layer(1024, types.OCIUncompressedRestrictedLayer)
		if err != nil {
			r.t.Fatalf("unable to create a layer %s", err)
		}
		r.state[imageName].image, err = mutate.AppendLayers(r.state[imageName].image, layer)
		if err != nil {
			r.t.Fatalf("unable to append a layer %s", err)
		}
	}
}

func (r *FakeRegistry) CleanUp() {
	for _, tarPath := range r.state {
		os.Remove(tarPath.path)
	}
}

type ImageWithTarPath struct {
	image v1.Image
	path  string
	t     *testing.T
}

func (r *ImageWithTarPath) WithNonDistributableLayer() {
	layer, err := random.Layer(1024, types.OCIUncompressedRestrictedLayer)
	if err != nil {
		r.t.Fatalf("unable to create a layer %s", err)
	}
	r.image, err = mutate.AppendLayers(r.image, layer)
	if err != nil {
		r.t.Fatalf("unable to append a layer %s", err)
	}
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
