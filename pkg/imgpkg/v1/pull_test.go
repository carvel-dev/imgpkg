// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
	v1 "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/v1"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

func TestPullImage(t *testing.T) {
	bundleName := "some/bundle"
	imageName := "some/image"
	imageTag := "some-image-tag"
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})

	img1 := fakeRegistry.WithRandomImage("some/image-1")
	img2 := fakeRegistry.WithRandomImage("some/image-2")
	randomBundle := createBundleWithImages(fakeRegistry, bundleName, []string{img1.RefDigest, img2.RefDigest})
	randomImg := fakeRegistry.WithRandomImage(imageName)
	uiLogger := util.NewNoopLevelLogger()
	fakeRegistry.Tag(randomImg.RefDigest, imageTag)

	defer fakeRegistry.CleanUp()
	fakeRegistry.Build()

	t.Run("succeeds when reference provided is image, not cacheable pulling by tag", func(t *testing.T) {
		outputFolder, err := os.MkdirTemp("", "imgpkg-v1-test")
		require.NoError(t, err)
		defer os.Remove(outputFolder)

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  true,
			IsBundle: false,
		}
		status, err := v1.Pull(fakeRegistry.ReferenceOnTestServer(imageName)+":"+imageTag, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.Equal(t, v1.PullStatus{
			BundleInfo: v1.BundleInfo{
				ImageRef: randomImg.RefDigest,
			},
			Cacheable: false,
			IsBundle:  false,
		}, status)
	})

	t.Run("is cacheable when pulling by digest", func(t *testing.T) {
		outputFolder, err := os.MkdirTemp("", "imgpkg-v1-test")
		require.NoError(t, err)
		defer os.Remove(outputFolder)

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  true,
			IsBundle: false,
		}
		status, err := v1.Pull(randomImg.RefDigest, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.Equal(t, v1.PullStatus{
			BundleInfo: v1.BundleInfo{
				ImageRef: randomImg.RefDigest,
			},
			Cacheable: true,
			IsBundle:  false,
		}, status)
	})

	t.Run("it succeeds when downloading the OCI Image of the bundle", func(t *testing.T) {
		outputFolder, err := os.MkdirTemp("", "imgpkg-v1-test")
		require.NoError(t, err)
		defer os.Remove(outputFolder)

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  true,
			IsBundle: false,
		}
		_, err = v1.Pull(randomBundle, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
	})

	t.Run("it fails when downloading the a bundle", func(t *testing.T) {
		outputFolder, err := os.MkdirTemp("", "imgpkg-v1-test")
		require.NoError(t, err)
		defer os.Remove(outputFolder)

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  false,
			IsBundle: false,
		}
		_, err = v1.Pull(randomBundle, outputFolder, opts, registry.Opts{})
		require.ErrorContains(t, err, "The provided image is a bundle")
	})
}

func TestPullBundle(t *testing.T) {
	bundleName := "some/bundle"
	collocatedBundle := "some/collocated-bundle"
	collocatedBundleTag := "colocated-bundle-tag"
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	img1 := fakeRegistry.WithRandomImage("some/image-1")
	img2 := fakeRegistry.WithRandomImage("some/image-2")
	randomBundle := createBundleWithImages(fakeRegistry, bundleName, []string{img1.RefDigest, img2.RefDigest})

	colImg1 := fakeRegistry.CopyImage(*img1, collocatedBundle)
	colImg2 := fakeRegistry.CopyImage(*img2, collocatedBundle)
	collocatedBundleRef := createBundleWithImages(fakeRegistry, collocatedBundle, []string{img1.RefDigest, img2.RefDigest})
	fakeRegistry.Tag(collocatedBundleRef, collocatedBundleTag)

	uiLogger := util.NewNoopLevelLogger()

	defer fakeRegistry.CleanUp()
	fakeRegistry.Build()

	t.Run("succeeds when reference provided is a bundle that was not copied and does not update ImagesLock file", func(t *testing.T) {
		outputFolder := t.TempDir()

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  false,
			IsBundle: true,
		}
		status, err := v1.Pull(randomBundle, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.Equal(t, v1.PullStatus{
			BundleInfo: v1.BundleInfo{
				ImageRef: randomBundle,
				ImagesLock: &v1.ImagesLockInfo{
					Path: filepath.Join(outputFolder, ".imgpkg", "images.yml"),
				},
			},
			Cacheable: false,
			IsBundle:  true,
		}, status)

		// Ensures that pulled ImagesLock file was not changed
		assertImagesLock(t, outputFolder, []string{img1.RefDigest, img2.RefDigest})
	})

	t.Run("succeeds when pulling the bundle OCI image and does not update ImagesLock file", func(t *testing.T) {
		outputFolder := t.TempDir()

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  true,
			IsBundle: true,
		}
		status, err := v1.Pull(randomBundle, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.Equal(t, v1.PullStatus{
			BundleInfo: v1.BundleInfo{
				ImageRef: randomBundle,
			},
			Cacheable: true,
			IsBundle:  true,
		}, status)

		assertImagesLock(t, outputFolder, []string{img1.RefDigest, img2.RefDigest})
	})

	t.Run("succeeds when reference provided is a bundle that was copied and updates ImagesLock file", func(t *testing.T) {
		outputFolder := t.TempDir()

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  false,
			IsBundle: true,
		}
		status, err := v1.Pull(collocatedBundleRef, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.Equal(t, v1.PullStatus{
			BundleInfo: v1.BundleInfo{
				ImageRef: collocatedBundleRef,
				ImagesLock: &v1.ImagesLockInfo{
					Path:    filepath.Join(outputFolder, ".imgpkg", "images.yml"),
					Updated: true,
				},
			},
			Cacheable: true,
			IsBundle:  true,
		}, status)

		// Ensures that pulled ImagesLock file is updated correctly
		assertImagesLock(t, outputFolder, []string{colImg1.RefDigest, colImg2.RefDigest})
	})

	t.Run("succeeds when pulling the bundle OCI image of collocated bundle does not update ImagesLock", func(t *testing.T) {
		outputFolder := t.TempDir()

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  true,
			IsBundle: true,
		}
		status, err := v1.Pull(collocatedBundleRef, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.Equal(t, v1.PullStatus{
			BundleInfo: v1.BundleInfo{
				ImageRef: collocatedBundleRef,
			},
			Cacheable: true,
			IsBundle:  true,
		}, status)

		assertImagesLock(t, outputFolder, []string{img1.RefDigest, img2.RefDigest})
	})

	t.Run("succeeds when pulling using the tag it returns not cacheable image", func(t *testing.T) {
		outputFolder := t.TempDir()
		ctag, err := regname.ParseReference(collocatedBundleRef)
		require.NoError(t, err)
		imgTag := ctag.Context().Tag(collocatedBundleTag)

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  true,
			IsBundle: true,
		}
		status, err := v1.Pull(imgTag.String(), outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.Equal(t, v1.PullStatus{
			BundleInfo: v1.BundleInfo{
				ImageRef: collocatedBundleRef,
			},
			Cacheable: false,
			IsBundle:  true,
		}, status)

		assertImagesLock(t, outputFolder, []string{img1.RefDigest, img2.RefDigest})
	})

	t.Run("fails, when image is not a bundle", func(t *testing.T) {
		outputFolder, err := os.MkdirTemp("", "imgpkg-v1-test")
		require.NoError(t, err)
		defer os.Remove(outputFolder)

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  false,
			IsBundle: true,
		}
		_, err = v1.Pull(img1.RefDigest, outputFolder, opts, registry.Opts{})
		require.ErrorContains(t, err, "The provided image is not a bundle")
	})
}

func TestPullBundleRecursively(t *testing.T) {
	simpleBundleName := "some/bundle"
	bundleWithNestedName := "some/outer-bundle"
	bundleTag := "bundle-tag"
	collocatedBundle := "some/collocated-bundle"
	collocatedBundleTag := "colocated-bundle-tag"
	fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
	img1 := fakeRegistry.WithRandomImage("some/image-1")
	img2 := fakeRegistry.WithRandomImage("some/image-2")
	simpleBundle := createBundle(fakeRegistry, simpleBundleName, []string{img1.RefDigest, img2.RefDigest})

	bundleWithNested := createBundleWithImages(fakeRegistry, bundleWithNestedName, []string{img1.RefDigest, simpleBundle.RefDigest})

	colImg1 := fakeRegistry.CopyImage(*img1, collocatedBundle)
	colImg2 := fakeRegistry.CopyImage(*img2, collocatedBundle)
	colSimpleBundle := fakeRegistry.CopyBundleImage(simpleBundle, fakeRegistry.ReferenceOnTestServer(collocatedBundle))
	collocatedBundleRef := createBundleWithImages(fakeRegistry, collocatedBundle, []string{img1.RefDigest, simpleBundle.RefDigest})
	fakeRegistry.Tag(collocatedBundleRef, collocatedBundleTag)
	fakeRegistry.Tag(bundleWithNested, bundleTag)

	uiLogger := util.NewNoopLevelLogger()

	defer fakeRegistry.CleanUp()
	fakeRegistry.Build()

	t.Run("succeeds when reference provided is a bundle that was not copied and does not update ImagesLock file", func(t *testing.T) {
		outputFolder := t.TempDir()

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  false,
			IsBundle: true,
		}
		status, err := v1.PullRecursive(bundleWithNested, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)

		digest, err := regname.NewDigest(simpleBundle.RefDigest)
		require.NoError(t, err)
		hash, err := regv1.NewHash(digest.DigestStr())
		require.NoError(t, err)
		expectedNestedBundlePath := filepath.Join(outputFolder, ".imgpkg", "bundles", fmt.Sprintf("%s-%s", hash.Algorithm, hash.Hex))

		require.Equal(t, v1.PullStatus{
			BundleInfo: v1.BundleInfo{
				ImageRef: bundleWithNested,
				ImagesLock: &v1.ImagesLockInfo{
					Path:    filepath.Join(outputFolder, ".imgpkg", "images.yml"),
					Updated: false,
				},
				NestedBundles: []v1.BundleInfo{{
					ImageRef: simpleBundle.RefDigest,
					ImagesLock: &v1.ImagesLockInfo{
						Path:    filepath.Join(expectedNestedBundlePath, ".imgpkg", "images.yml"),
						Updated: false,
					},
				}},
			},
			Cacheable: false,
			IsBundle:  true,
		}, status)

		// Ensures that pulled ImagesLock file was not changed
		assertImagesLock(t, outputFolder, []string{img1.RefDigest, simpleBundle.RefDigest})
		assertImagesLock(t, filepath.Join(expectedNestedBundlePath), []string{img1.RefDigest, img2.RefDigest})
	})

	t.Run("when bundle is not fully collocated it is not cacheable", func(t *testing.T) {
		outputFolder := t.TempDir()

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  false,
			IsBundle: true,
		}
		status, err := v1.PullRecursive(bundleWithNested, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.False(t, status.Cacheable)
	})

	t.Run("when bundle referenced by tag and is NOT fully collocated it is NOT cacheable", func(t *testing.T) {
		outputFolder := t.TempDir()
		ctag, err := regname.ParseReference(bundleWithNested)
		require.NoError(t, err)
		imgTag := ctag.Context().Tag(bundleTag)

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  false,
			IsBundle: true,
		}
		status, err := v1.PullRecursive(imgTag.String(), outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.False(t, status.Cacheable)
	})

	t.Run("succeeds when reference provided is a bundle that was copied and does update ImagesLock file", func(t *testing.T) {
		outputFolder := t.TempDir()

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  false,
			IsBundle: true,
		}
		status, err := v1.PullRecursive(collocatedBundleRef, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)

		digest, err := regname.NewDigest(simpleBundle.RefDigest)
		require.NoError(t, err)
		hash, err := regv1.NewHash(digest.DigestStr())
		require.NoError(t, err)
		expectedNestedBundlePath := filepath.Join(outputFolder, ".imgpkg", "bundles", fmt.Sprintf("%s-%s", hash.Algorithm, hash.Hex))
		colBundleDigest, err := regname.NewDigest(collocatedBundleRef)
		require.NoError(t, err)
		collocatedSimpleBundleRef := colBundleDigest.Digest(digest.DigestStr())

		require.Equal(t, v1.PullStatus{
			BundleInfo: v1.BundleInfo{
				ImageRef: collocatedBundleRef,
				ImagesLock: &v1.ImagesLockInfo{
					Path:    filepath.Join(outputFolder, ".imgpkg", "images.yml"),
					Updated: true,
				},
				NestedBundles: []v1.BundleInfo{{
					ImageRef: collocatedSimpleBundleRef.String(),
					ImagesLock: &v1.ImagesLockInfo{
						Path:    filepath.Join(expectedNestedBundlePath, ".imgpkg", "images.yml"),
						Updated: true,
					},
				}},
			},
			Cacheable: true,
			IsBundle:  true,
		}, status)

		// Ensures that pulled ImagesLock file was changed
		assertImagesLock(t, outputFolder, []string{colImg1.RefDigest, colSimpleBundle.RefDigest})
		assertImagesLock(t, filepath.Join(expectedNestedBundlePath), []string{colImg1.RefDigest, colImg2.RefDigest})
	})

	t.Run("when bundle is fully collocated it is cacheable", func(t *testing.T) {
		outputFolder := t.TempDir()

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  false,
			IsBundle: true,
		}
		status, err := v1.PullRecursive(collocatedBundleRef, outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.True(t, status.Cacheable)
	})

	t.Run("when bundle referenced by tag and is fully collocated it is NOT cacheable", func(t *testing.T) {
		outputFolder := t.TempDir()
		ctag, err := regname.ParseReference(collocatedBundleRef)
		require.NoError(t, err)
		imgTag := ctag.Context().Tag(collocatedBundleTag)

		opts := v1.PullOpts{
			Logger:   uiLogger,
			AsImage:  false,
			IsBundle: true,
		}
		status, err := v1.PullRecursive(imgTag.String(), outputFolder, opts, registry.Opts{})
		require.NoError(t, err)
		require.False(t, status.Cacheable)
	})
}

func assertImagesLock(t *testing.T, outputFolder string, expectedImagesRefs []string) {
	imagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(outputFolder, ".imgpkg", "images.yml"))
	require.NoError(t, err)
	require.Len(t, imagesLock.Images, 2)
	for i, imgRef := range expectedImagesRefs {
		got := imagesLock.Images[i]
		assert.Equalf(t, imgRef, got.Image, "image %d", i)
	}
}

func createBundle(fakeRegistry *helpers.FakeTestRegistryBuilder, bundleName string, refs []string) helpers.BundleInfo {
	randomBundle := fakeRegistry.WithRandomBundle(bundleName)
	var imgs []lockconfig.ImageRef
	for _, ref := range refs {
		imgs = append(imgs, lockconfig.ImageRef{
			Image: ref,
		})
	}
	return randomBundle.WithImageRefs(imgs)
}

func createBundleWithImages(fakeRegistry *helpers.FakeTestRegistryBuilder, bundleName string, refs []string) string {
	return createBundle(fakeRegistry, bundleName, refs).RefDigest
}
