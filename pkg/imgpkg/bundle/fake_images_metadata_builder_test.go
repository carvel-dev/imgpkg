// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagelayers"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/stretchr/testify/assert"
)

type FakeTestRegistryBuilder struct {
	images map[string]*ImageWithTarPath
	server *httptest.Server
	t      *testing.T
}

func NewFakeImagesMetadataBuilder(t *testing.T) *FakeTestRegistryBuilder {
	r := &FakeTestRegistryBuilder{images: map[string]*ImageWithTarPath{}, t: t}
	r.server = httptest.NewServer(registry.New())

	return r
}

func (r *FakeTestRegistryBuilder) Build() image.ImagesMetadata {
	u, err := url.Parse(r.server.URL)
	assert.NoError(r.t, err)

	for imageRef, val := range r.images {
		imageRefWithTestRegistry, err := name.ParseReference(fmt.Sprintf("%s/%s", u.Host, imageRef))
		assert.NoError(r.t, err)
		err = regremote.Write(imageRefWithTestRegistry, val.image)
		assert.NoError(r.t, err)
	}

	reg, err := image.NewRegistry(image.RegistryOpts{}, imagelayers.ImageLayerWriterFilter{})
	assert.NoError(r.t, err)
	return reg
}

func (r *FakeTestRegistryBuilder) WithBundleFromPath(bundleName string, path string) BundleInfo {
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
	digest, err := bundle.Digest()
	if err != nil {
		r.t.Fatalf(err.Error())
	}

	return BundleInfo{r, bundleName, path, digest.String()}
}

func (r *FakeTestRegistryBuilder) WithImageFromPath(imageNameFromTest string, path string, labels map[string]string) *ImageWithTarPath {
	tarballLayer, err := compress(path)
	if err != nil {
		r.t.Fatalf("Failed trying to compress %s: %s", path, err)
	}

	fileImage, err := image.NewFileImage(tarballLayer.Name(), labels)
	if err != nil {
		r.t.Fatalf("Failed trying to build a file image%s", err)
	}

	return r.updateState(imageNameFromTest, fileImage, path)
}

func (r *FakeTestRegistryBuilder) CleanUp() {
	for _, tarPath := range r.images {
		os.Remove(filepath.Join(tarPath.path, ".imgpkg", "images.yml"))
	}
	if r.server != nil {
		r.server.Close()
	}
}

func (r *FakeTestRegistryBuilder) ReferenceOnTestServer(repo string) string {
	u, err := url.Parse(r.server.URL)
	assert.NoError(r.t, err)
	return fmt.Sprintf("%s/%s", u.Host, repo)
}

func (r *FakeTestRegistryBuilder) updateState(imageName string, image v1.Image, path string) *ImageWithTarPath {
	imgName, err := name.ParseReference(imageName)
	if err != nil {
		r.t.Fatalf("unable to parse reference: %s", err)
	}

	imageOrImageIndexWithTarPath := &ImageWithTarPath{fakeRegistry: r, t: r.t, image: image, path: path}
	r.images[imgName.Context().RepositoryStr()] = imageOrImageIndexWithTarPath
	return imageOrImageIndexWithTarPath
}

type BundleInfo struct {
	r          *FakeTestRegistryBuilder
	BundleName string
	BundlePath string
	Digest     string
}

func (b BundleInfo) WithEveryImageFrom(path string, labels map[string]string) BundleInfo {
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
		assert.NoError(b.r.t, err)

		u, err := url.Parse(b.r.server.URL)
		assert.NoError(b.r.t, err)
		imageRefs = append(imageRefs, lockconfig.ImageRef{
			Image: u.Host + "/" + imageRef.Context().RepositoryStr() + "@" + digest.String(),
		})
	}

	imagesLock.Images = imageRefs
	err = imagesLock.WriteToPath(filepath.Join(b.BundlePath, bundle.ImgpkgDir, bundle.ImagesLockFile))
	if err != nil {
		b.r.t.Fatalf("Got error: %s", err.Error())
	}

	return b.r.WithBundleFromPath(b.BundleName, b.BundlePath)
}

type ImageWithTarPath struct {
	fakeRegistry *FakeTestRegistryBuilder
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
