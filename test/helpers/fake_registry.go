// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

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
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagelayers"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FakeTestRegistryBuilder struct {
	images map[string]*ImageOrImageIndexWithTarPath
	server *httptest.Server
	t      *testing.T
}

func NewFakeRegistry(t *testing.T) *FakeTestRegistryBuilder {
	r := &FakeTestRegistryBuilder{images: map[string]*ImageOrImageIndexWithTarPath{}, t: t}
	r.server = httptest.NewServer(registry.New())

	return r
}

func (r *FakeTestRegistryBuilder) Build() image.Registry {
	u, err := url.Parse(r.server.URL)
	assert.NoError(r.t, err)

	for imageRef, val := range r.images {
		imageRefWithTestRegistry, err := name.ParseReference(fmt.Sprintf("%s/%s", u.Host, imageRef))
		assert.NoError(r.t, err)

		if val.Image != nil {
			err = regremote.Write(imageRefWithTestRegistry, val.Image, regremote.WithNondistributable)
			assert.NoError(r.t, err)
			err = regremote.Tag(imageRefWithTestRegistry.Context().Tag("latest"), val.Image)
			assert.NoError(r.t, err)
		}

		if val.imageIndex != nil {
			err = regremote.WriteIndex(imageRefWithTestRegistry, val.imageIndex, regremote.WithNondistributable)
			assert.NoError(r.t, err)
			err = regremote.Tag(imageRefWithTestRegistry.Context().Tag("latest"), val.imageIndex)
			assert.NoError(r.t, err)
		}

	}

	reg, err := image.NewRegistry(image.RegistryOpts{}, imagelayers.NewImageLayerWriterCheck(false))
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

	r.updateState(bundleName, bundle, nil, path)
	digest, err := bundle.Digest()
	assert.NoError(r.t, err)

	return BundleInfo{r, bundleName, path, digest.String(), r.ReferenceOnTestServer(bundleName + "@" + digest.String())}
}

func (r *FakeTestRegistryBuilder) WithRandomBundle(bundleName string) BundleInfo {
	bundle, err := random.Image(500, 5)
	bundle, err = mutate.ConfigFile(bundle, &v1.ConfigFile{
		Config: v1.Config{
			Labels: map[string]string{"dev.carvel.imgpkg.bundle": "true"},
		},
	})
	require.NoError(r.t, err, "create image from tar")

	r.updateState(bundleName, bundle, nil, "")

	digest, err := bundle.Digest()
	assert.NoError(r.t, err)

	return BundleInfo{r, bundleName, "", digest.String(), r.ReferenceOnTestServer(bundleName + "@" + digest.String())}
}

func (r *FakeTestRegistryBuilder) WithImageFromPath(imageNameFromTest string, path string, labels map[string]string) *ImageOrImageIndexWithTarPath {
	tarballLayer, err := compress(path)
	if err != nil {
		r.t.Fatalf("Failed trying to compress %s: %s", path, err)
	}

	fileImage, err := image.NewFileImage(tarballLayer.Name(), labels)
	if err != nil {
		r.t.Fatalf("Failed trying to build a file image%s", err)
	}

	return r.updateState(imageNameFromTest, fileImage, nil, path)
}

func (r *FakeTestRegistryBuilder) WithRandomImage(imageNameFromTest string) *ImageOrImageIndexWithTarPath {
	img, err := random.Image(500, 3)
	require.NoError(r.t, err, "create image from tar")

	return r.updateState(imageNameFromTest, img, nil, "")
}

func (r *FakeTestRegistryBuilder) WithImage(imageNameFromTest string, image v1.Image) *ImageOrImageIndexWithTarPath {
	return r.updateState(imageNameFromTest, image, nil, "")
}

func (r *FakeTestRegistryBuilder) CopyImage(img ImageOrImageIndexWithTarPath, to string) *ImageOrImageIndexWithTarPath {
	return r.updateState(to, img.Image, nil, "")
}

func (r *FakeTestRegistryBuilder) CopyBundleImage(bundleInfo BundleInfo, to string) BundleInfo {
	newBundle := *r.images[bundleInfo.BundleName]
	r.updateState(to, newBundle.Image, nil, "")
	return BundleInfo{r, to, "", bundleInfo.Digest, bundleInfo.RefDigest}
}

func (r *FakeTestRegistryBuilder) WithARandomImageIndex(imageName string) {
	index, err := random.Index(1024, 1, 1)
	require.NoError(r.t, err)

	r.updateState(imageName, nil, index, "")
}

func (r *FakeTestRegistryBuilder) WithNonDistributableLayerInImage(imageNames ...string) {
	for _, imageName := range imageNames {
		layer, err := random.Layer(1024, types.OCIUncompressedRestrictedLayer)
		require.NoErrorf(r.t, err, "create layer: %s", imageName)

		imageWithARestrictedLayer, err := mutate.AppendLayers(r.images[imageName].Image, layer)
		require.NoErrorf(r.t, err, "add layer: %s", imageName)

		r.updateState(imageName, imageWithARestrictedLayer, r.images[imageName].imageIndex, r.images[imageName].path)
	}
}

func (r *ImageOrImageIndexWithTarPath) WithNonDistributableLayer() *ImageOrImageIndexWithTarPath {
	layer, err := random.Layer(1024, types.OCIUncompressedRestrictedLayer)
	require.NoError(r.t, err)

	r.Image, err = mutate.AppendLayers(r.Image, layer)
	require.NoError(r.t, err)
	return r.fakeRegistry.updateState(r.RefDigest, r.Image, r.imageIndex, r.path)
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

func (r *FakeTestRegistryBuilder) updateState(imageName string, image v1.Image, imageIndex v1.ImageIndex, path string) *ImageOrImageIndexWithTarPath {
	imgName, err := name.ParseReference(imageName)
	require.NoError(r.t, err)

	imageOrImageIndexWithTarPath := &ImageOrImageIndexWithTarPath{fakeRegistry: r, t: r.t, Image: image, imageIndex: imageIndex, path: path}
	if image != nil {
		digest, err := image.Digest()
		require.NoError(r.t, err)
		imgName, err := name.ParseReference(imageName)
		require.NoError(r.t, err)
		imageOrImageIndexWithTarPath.RefDigest = r.ReferenceOnTestServer(imgName.Context().RepositoryStr() + "@" + digest.String())
		r.images[imgName.Context().RepositoryStr()+"@"+digest.String()] = imageOrImageIndexWithTarPath
	} else {
		r.images[imgName.Context().RepositoryStr()] = imageOrImageIndexWithTarPath
	}

	return imageOrImageIndexWithTarPath
}

type BundleInfo struct {
	r          *FakeTestRegistryBuilder
	BundleName string
	BundlePath string
	Digest     string
	RefDigest  string
}

func (b BundleInfo) WithEveryImageFromPath(path string, labels map[string]string) BundleInfo {
	imgLockPath := filepath.Join(b.BundlePath, ".imgpkg", "images.yml.template")
	imgLock, err := lockconfig.NewImagesLockFromPath(imgLockPath)
	assert.NoError(b.r.t, err)

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
		assert.NoError(b.r.t, err)

		digest, err := imageFromPath.Image.Digest()
		assert.NoError(b.r.t, err)

		u, err := url.Parse(b.r.server.URL)
		assert.NoError(b.r.t, err)
		imageRefs = append(imageRefs, lockconfig.ImageRef{
			Image: u.Host + "/" + imageRef.Context().RepositoryStr() + "@" + digest.String(),
		})
	}

	imagesLock.Images = imageRefs
	imagesLockFile := filepath.Join(b.BundlePath, bundle.ImgpkgDir, bundle.ImagesLockFile)
	err = imagesLock.WriteToPath(imagesLockFile)
	assert.NoError(b.r.t, err)

	delete(b.r.images, b.BundleName+"@"+b.Digest)
	return b.r.WithBundleFromPath(b.BundleName, b.BundlePath)
}

func (b BundleInfo) WithImageRefs(imageRefs []lockconfig.ImageRef) BundleInfo {
	imagesLock := lockconfig.ImagesLock{
		LockVersion: lockconfig.LockVersion{
			APIVersion: lockconfig.ImagesLockAPIVersion,
			Kind:       lockconfig.ImagesLockKind,
		},
	}

	imagesLock.Images = imageRefs
	err := imagesLock.WriteToPath(filepath.Join(b.BundlePath, bundle.ImgpkgDir, bundle.ImagesLockFile))
	assert.NoError(b.r.t, err)

	delete(b.r.images, b.BundleName+"@"+b.Digest)
	return b.r.WithBundleFromPath(b.BundleName, b.BundlePath)
}

type ImageOrImageIndexWithTarPath struct {
	fakeRegistry *FakeTestRegistryBuilder
	Image        v1.Image
	imageIndex   v1.ImageIndex
	path         string
	t            *testing.T
	RefDigest    string
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
