// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

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
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/image/imagefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

type FakeRegistry struct {
	state map[string]*ImageWithTarPath
	t     *testing.T
}

func NewFakeRegistry(t *testing.T) *FakeRegistry {
	return &FakeRegistry{state: map[string]*ImageWithTarPath{}, t: t}
}

func (r *FakeRegistry) Build() *imagefakes.FakeImagesMetadata {
	fakeRegistry := &imagefakes.FakeImagesMetadata{}
	getDescriptor := func(reference name.Reference, r *FakeRegistry) (v1.Descriptor, error) {
		if val, found := r.state[reference.Name()]; found {
			if val.image != nil {
				mediaType, err := val.image.MediaType()
				digest, err := val.image.Digest()
				if err != nil {
					r.t.Fatal(err.Error())
				}
				return v1.Descriptor{
					MediaType: mediaType,
					Digest:    digest,
				}, nil
			}
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
		}

		return &regremote.Descriptor{}, fmt.Errorf("FakeRegistry: GetCall: image [%s] not found", reference.Name())
	})

	fakeRegistry.ImageStub = func(reference name.Reference) (v v1.Image, err error) {
		if bundle, found := r.state[reference.Name()]; found {
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
	if err != nil {
		r.t.Fatalf("unable to create image from file: %s", err)
	}

	r.updateState(bundleName, bundle, path)
	return BundleInfo{r, bundleName, path}
}

func (r *FakeRegistry) WithImageFromPath(imageNameFromTest string, path string, labels map[string]string) *ImageWithTarPath {
	tarballLayer, err := compress(path)
	if err != nil {
		r.t.Fatalf("Failed trying to compress %s: %s", path, err)
	}

	fileImage, err := image.NewFileImage(tarballLayer.Name(), labels)
	if err != nil {
		r.t.Fatalf("Failed trying to build a file image%s", err)
	}

	r.updateState(imageNameFromTest, fileImage, path)
	reference, err := name.ParseReference(imageNameFromTest)
	if err != nil {
		r.t.Fatalf("Failed trying to get image name: %s", err)
	}

	return r.state[reference.Name()]
}

func (r *FakeRegistry) CleanUp() {
	for _, tarPath := range r.state {
		os.Remove(filepath.Join(tarPath.path, ".imgpkg", "images.yml"))
	}
}

func (r *FakeRegistry) updateState(imageName string, image v1.Image, path string) {
	imgName, err := name.ParseReference(imageName)
	if err != nil {
		r.t.Fatalf("unable to parse reference: %s", err)
	}

	imageOrImageIndexWithTarPath := &ImageWithTarPath{fakeRegistry: r, t: r.t, imageName: imageName, image: image, path: path}
	r.state[imgName.Name()] = imageOrImageIndexWithTarPath

	if image != nil {
		digest, err := image.Digest()
		if err != nil {
			r.t.Fatalf("unable to parse reference: %s", err)
		}
		r.state[imgName.Context().Name()+"@"+digest.String()] = imageOrImageIndexWithTarPath
	}
}

type BundleInfo struct {
	r          *FakeRegistry
	BundleName string
	BundlePath string
}

func (b BundleInfo) WithEveryImageFrom(path string, labels map[string]string) *FakeRegistry {
	imgLockPath := filepath.Join(b.BundlePath, ".imgpkg", "images.yml.template")
	imgLock, err := lockconfig.NewImagesLockFromPath(imgLockPath)
	if err != nil {
		b.r.t.Fatalf("Got error: %s", err.Error())
	}

	var imageRefs []lockconfig.ImageRef
	imagesLock := lockconfig.ImagesLock{
		LockVersion: lockconfig.LockVersion{
			APIVersion: lockconfig.ImagesLockAPIVersion,
			Kind:       lockconfig.ImagesLockKind,
		},
	}

	for _, img := range imgLock.Images {
		imageFromPath := b.r.WithImageFromPath(img.Image, path, labels)
		imageRef, err := name.ParseReference(img.Image)
		if err != nil {
			b.r.t.Fatalf("Got error: %s", err.Error())
		}

		digest, err := imageFromPath.image.Digest()
		if err != nil {
			b.r.t.Fatalf("Got error: %s", err.Error())
		}
		imageRefs = append(imageRefs, lockconfig.ImageRef{
			Image: imageRef.Context().RepositoryStr() + "@" + digest.String(),
		})
	}

	imagesLock.Images = imageRefs
	err = imagesLock.WriteToPath(filepath.Join(b.BundlePath, bundle.ImgpkgDir, bundle.ImagesLockFile))
	if err != nil {
		b.r.t.Fatalf("Got error: %s", err.Error())
	}

	b.r.WithBundleFromPath(b.BundleName, b.BundlePath)
	return b.r
}

type ImageWithTarPath struct {
	fakeRegistry *FakeRegistry
	imageName    string
	image        v1.Image
	path         string
	t            *testing.T
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
