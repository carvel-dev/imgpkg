// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	regname "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

func TestCopyBundleToDifferentRepository(t *testing.T) {
	logger := &helpers.Logger{}

	t.Run("when all images are collocated it copies all images from the original location and generate a BundleLock", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
		var bundleDigest, imageDigest string
		logger.Section("create bundle with image", func() {
			imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
`, env.Image, imageDigest)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s%s", env.Image, bundleTag), "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to repository and generate BundleLock", func() {
			lockOutputPath := filepath.Join(env.Assets.CreateTempFolder("bundle-lock"), "bundle-relocate-lock.yml")
			imgpkg.Run([]string{"copy",
				"--bundle", fmt.Sprintf("%s%s", env.Image, bundleTag),
				"--to-repo", env.RelocationRepo,
				"--lock-output", lockOutputPath},
			)

			expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
			expectedTag := strings.TrimPrefix(bundleTag, ":")
			env.Assert.AssertBundleLock(lockOutputPath, expectedRef, expectedTag)
		})

		refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleTag, env.RelocationRepo + bundleDigest}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
	})

	t.Run("when copying bundle it adds a default tag to all images", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
		var bundleDigest, imageDigest string
		logger.Section("create bundle with image", func() {
			imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
`, env.Image, imageDigest)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s%s", env.Image, bundleTag), "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to repository", func() {
			imgpkg.Run([]string{"copy",
				"--bundle", fmt.Sprintf("%s%s", env.Image, bundleTag),
				"--to-repo", env.RelocationRepo},
			)
		})

		logger.Section("Check default tag was created in the bundle and all the images", func() {
			algorithmAndSHA := strings.Split(imageDigest, "@")[1]
			splitAlgAndSHA := strings.Split(algorithmAndSHA, ":")
			imageDefaultTag := fmt.Sprintf("%s:%s-%s.imgpkg", env.RelocationRepo, splitAlgAndSHA[0], splitAlgAndSHA[1])

			algorithmAndSHA = strings.Split(bundleDigest, "@")[1]
			splitAlgAndSHA = strings.Split(algorithmAndSHA, ":")
			bundleDefaultTag := fmt.Sprintf("%s:%s-%s.imgpkg", env.RelocationRepo, splitAlgAndSHA[0], splitAlgAndSHA[1])
			require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry([]string{imageDefaultTag, bundleDefaultTag}))
		})
	})

	t.Run("when some images are not in the same repository as the bundle it copies all images", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		image := env.Image + "-image-outside-repo"
		var imageDigest, bundleDigestRef, bundleDigest string
		logger.Section("create bundle with image", func() {
			imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, image)
			// image intentionally does not exist in bundle repo
			imageDigestRef := image + imageDigest

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			bundleDigestRef = env.Image + bundleDigest
		})

		imgpkg.Run([]string{"copy", "--bundle", bundleDigestRef, "--to-repo", env.RelocationRepo})

		refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
	})

	t.Run("when copying bundle with annotations in ImagesLock it maintain the annotation after the copy", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		var bundleDigestRef, bundleDigest string
		logger.Section("create bundle with image", func() {
			imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
			imageDigestRef := env.Image + imageDigest

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
  annotations:
    greeting: hello world
`, imageDigestRef)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			bundleDigestRef = env.Image + bundleDigest
		})

		logger.Section("copy bundle", func() {
			imgpkg.Run([]string{"copy", "-b", bundleDigestRef, "--to-repo", env.RelocationRepo})
		})

		testDir := env.Assets.CreateTempFolder("test-annotation")
		logger.Section("pull bundle to read the ImagesLock file", func() {
			bundleDigestRef = env.RelocationRepo + bundleDigest
			imgpkg.Run([]string{"pull", "-b", bundleDigestRef, "-o", testDir})
		})

		imgLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(testDir, ".imgpkg", "images.yml"))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"greeting": "hello world"}, imgLock.Images[0].Annotations)
	})

	t.Run("when bundle contains a bundle it copies all the bundle's images from the original location", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest string
		logger.Section("create 2 simple images", func() {

			img1DigestRef = imgRef.Context().Name() + "-img1"
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img1DigestRef)
			img1DigestRef = img1DigestRef + img1Digest

			img2DigestRef = imgRef.Context().Name() + "-img2"
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img2DigestRef)
			img2DigestRef = img2DigestRef + img2Digest
		})

		simpleBundle := imgRef.Context().Name() + "-simple-bundle"
		simpleBundleDigest := ""
		logger.Section("create simple bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", simpleBundle, "-f", bundleDir})
			simpleBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		nestedBundle := imgRef.Context().Name() + "-bundle-nested"
		nestedBundleDigest := ""
		logger.Section("create nested bundle that contains images and the simple bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
- image: %s
`, img1DigestRef, img2DigestRef, simpleBundle+simpleBundleDigest)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", nestedBundle, "-f", bundleDir})
			nestedBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		outerBundle := imgRef.Context().Name() + "-bundle-outer"
		outerBundleDigest := ""
		bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
		bundleToCopy := fmt.Sprintf("%s%s", outerBundle, bundleTag)
		logger.Section("create outer bundle with image, simple bundle and nested bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
- image: %s
`, nestedBundle+nestedBundleDigest, img1DigestRef, simpleBundle+simpleBundleDigest)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", bundleToCopy, "-f", bundleDir})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to repository", func() {
			imgpkg.Run([]string{"copy",
				"--bundle", bundleToCopy,
				"--to-repo", env.RelocationRepo,
			},
			)
		})

		refs := []string{
			env.RelocationRepo + img1Digest,
			env.RelocationRepo + img2Digest,
			env.RelocationRepo + simpleBundleDigest,
			env.RelocationRepo + nestedBundleDigest,
			env.RelocationRepo + bundleTag,
			env.RelocationRepo + outerBundleDigest,
		}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
	})

	t.Run("when bundle contains a bundle with signed images it copies signatures", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest string
		logger.Section("create 2 simple images", func() {
			img1DigestRef = imgRef.Context().Name() + "-img1"
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img1DigestRef)
			img1DigestRef = img1DigestRef + img1Digest
			env.ImageFactory.SignImage(img1DigestRef)

			img2DigestRef = imgRef.Context().Name() + "-img2"
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img2DigestRef)
			img2DigestRef = img2DigestRef + img2Digest
			env.ImageFactory.SignImage(img2DigestRef)
		})

		simpleBundle := imgRef.Context().Name() + "-simple-bundle"
		var simpleBundleDigest string
		logger.Section("create simple bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", simpleBundle, "-f", bundleDir})
			simpleBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			env.ImageFactory.SignImage(simpleBundle + simpleBundleDigest)
		})

		nestedBundle := imgRef.Context().Name() + "-bundle-nested"
		var nestedBundleDigest string
		logger.Section("create nested bundle that contains images and the simple bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
- image: %s
`, img1DigestRef, img2DigestRef, simpleBundle+simpleBundleDigest)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", nestedBundle, "-f", bundleDir})
			nestedBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			env.ImageFactory.SignImage(nestedBundle + nestedBundleDigest)
		})

		outerBundle := imgRef.Context().Name() + "-bundle-outer"
		var outerBundleDigest string
		logger.Section("create outer bundle with image, simple bundle and nested bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
- image: %s
`, nestedBundle+nestedBundleDigest, img1DigestRef, simpleBundle+simpleBundleDigest)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", outerBundle, "-f", bundleDir})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			env.ImageFactory.SignImage(outerBundle + outerBundleDigest)
		})

		logger.Section("copy bundle to repository", func() {
			imgpkg.Run([]string{"copy",
				"--bundle", outerBundle + outerBundleDigest,
				"--to-repo", env.RelocationRepo,
				"--cosign-signatures",
			},
			)
		})

		refs := []string{
			env.RelocationRepo + ":" + img1Digest,
			env.RelocationRepo + ":" + img2Digest,
			env.RelocationRepo + ":" + simpleBundleDigest,
			env.RelocationRepo + ":" + nestedBundleDigest,
			env.RelocationRepo + ":" + outerBundleDigest,
		}
		env.Assert.ValidateCosignSignature(refs)
	})

	t.Run("when bundle is created in auth registry, copy the bundle to a public registry, after without credentials try to copy the bundle from the public registry", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()
		fakeRegistryBuilder := helpers.NewFakeRegistry(t, logger)
		const (
			username = "some-user"
			password = "some-password"
		)
		fakeRegistryBuilder.WithBasicAuth(username, password)
		_ = fakeRegistryBuilder.Build()
		defer fakeRegistryBuilder.CleanUp()

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest string
		logger.Section("create 2 simple images in auth registry", func() {
			img1DigestRef = fakeRegistryBuilder.ReferenceOnTestServer(imgRef.Context().RepositoryStr() + "-img1")
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFileWithAuth(imgpkg, img1DigestRef, fakeRegistryBuilder.Host(), username, password)
			img1DigestRef = img1DigestRef + img1Digest
			logger.Debugf("Created image: %s\n", img1DigestRef)

			img2DigestRef = fakeRegistryBuilder.ReferenceOnTestServer(imgRef.Context().RepositoryStr() + "-img2")
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFileWithAuth(imgpkg, img2DigestRef, fakeRegistryBuilder.Host(), username, password)
			img2DigestRef = img2DigestRef + img2Digest
			logger.Debugf("Created image: %s\n", img2DigestRef)
		})

		bundle := fakeRegistryBuilder.ReferenceOnTestServer(imgRef.Context().RepositoryStr() + "-bundle")
		bundleDigest := ""
		logger.Section("create nested bundle that contains images", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
`, img1DigestRef, img2DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out, err := imgpkg.RunWithOpts([]string{"push", "--tty", "-b", bundle, "-f", bundleDir}, helpers.RunOpts{
				EnvVars: []string{
					"IMGPKG_REGISTRY_HOSTNAME=" + fakeRegistryBuilder.Host(),
					"IMGPKG_REGISTRY_USERNAME=" + username,
					"IMGPKG_REGISTRY_PASSWORD=" + password},
			})
			require.NoError(t, err)
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			logger.Debugf("Created bundle: %s\n", bundle+bundleDigest)
		})

		relocatedBundle := env.RelocationRepo + "-bundle"
		logger.Section("copy bundle to repository from the private registry to the public registry", func() {
			out, err := imgpkg.RunWithOpts([]string{"copy",
				"--bundle", bundle + bundleDigest,
				"--to-repo", relocatedBundle,
			}, helpers.RunOpts{
				EnvVars: []string{
					"IMGPKG_REGISTRY_HOSTNAME=" + fakeRegistryBuilder.Host(),
					"IMGPKG_REGISTRY_USERNAME=" + username,
					"IMGPKG_REGISTRY_PASSWORD=" + password},
			})
			require.NoError(t, err)
			fmt.Println(out)
		})

		logger.Section("copy bundle from the public registry to a different repository", func() {
			imgpkg.Run([]string{"copy",
				"--bundle", relocatedBundle + bundleDigest,
				"--to-repo", relocatedBundle + "-copied",
			})
		})

		refs := []string{
			relocatedBundle + "-copied" + img1Digest,
			relocatedBundle + "-copied" + img2Digest,
			relocatedBundle + "-copied" + bundleDigest,
		}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
	})

	t.Run("When copy a simple bundle is preformed it generates image with locations", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
		var bundleDigest, imageDigest string
		logger.Section("create bundle with image", func() {
			imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
`, env.Image, imageDigest)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s%s", env.Image, bundleTag), "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to repository and generate BundleLock", func() {
			lockOutputPath := filepath.Join(env.Assets.CreateTempFolder("bundle-lock"), "bundle-relocate-lock.yml")
			imgpkg.Run([]string{"copy",
				"--bundle", fmt.Sprintf("%s%s", env.Image, bundleTag),
				"--to-repo", env.RelocationRepo,
				"--lock-output", lockOutputPath},
			)
		})

		hash, err := v1.NewHash(bundleDigest[1:])
		require.NoError(t, err)
		locationImg := fmt.Sprintf("%s:%s-%s.image-locations.imgpkg", env.RelocationRepo, hash.Algorithm, hash.Hex)

		logger.Section("check locations image was created", func() {
			refs := []string{locationImg}
			require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
		})

		logger.Section("download the locations file and check it", func() {
			locationImgFolder := env.Assets.CreateTempFolder("locations-img")
			env.ImageFactory.Download(locationImg, locationImgFolder)

			locationsFilePath := filepath.Join(locationImgFolder, "image-locations.yml")
			require.FileExists(t, locationsFilePath)

			cfg, err := bundle.NewLocationConfigFromPath(locationsFilePath)
			require.NoError(t, err)

			require.Equal(t, bundle.ImageLocationsConfig{
				APIVersion: "imgpkg.carvel.dev/v1alpha1",
				Kind:       "ImageLocations",
				Images: []bundle.ImageLocation{{
					Image: env.Image + imageDigest,
					// Repository not used for now because all images will be present in the same repository
					IsBundle: false,
				}},
			}, cfg)
		})
	})

	t.Run("when copy a bundle that contains a bundle it generates image with locations", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest string
		logger.Section("create 2 simple images", func() {

			img1DigestRef = imgRef.Context().Name() + "-img1"
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img1DigestRef)
			img1DigestRef = img1DigestRef + img1Digest

			img2DigestRef = imgRef.Context().Name() + "-img2"
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img2DigestRef)
			img2DigestRef = img2DigestRef + img2Digest
		})

		simpleBundle := imgRef.Context().Name() + "-simple-bundle"
		simpleBundleDigest := ""
		logger.Section("create simple bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", simpleBundle, "-f", bundleDir})
			simpleBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		nestedBundle := imgRef.Context().Name() + "-bundle-nested"
		nestedBundleDigest := ""
		logger.Section("create nested bundle that contains images and the simple bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
- image: %s
`, img1DigestRef, img2DigestRef, simpleBundle+simpleBundleDigest)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", nestedBundle, "-f", bundleDir})
			nestedBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		outerBundle := imgRef.Context().Name() + "-bundle-outer"
		outerBundleDigest := ""
		bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
		bundleToCopy := fmt.Sprintf("%s%s", outerBundle, bundleTag)
		logger.Section("create outer bundle with image, simple bundle and nested bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
- image: %s
`, nestedBundle+nestedBundleDigest, img1DigestRef, simpleBundle+simpleBundleDigest)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", bundleToCopy, "-f", bundleDir})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to repository", func() {
			imgpkg.Run([]string{"copy",
				"--bundle", bundleToCopy,
				"--to-repo", env.RelocationRepo,
			},
			)
		})

		logger.Section("download the locations file for outer bundle and check it", func() {
			downloadAndCheckLocationsFile(t, env, outerBundleDigest[1:], bundle.ImageLocationsConfig{
				APIVersion: "imgpkg.carvel.dev/v1alpha1",
				Kind:       "ImageLocations",
				Images: []bundle.ImageLocation{
					{
						Image:    nestedBundle + nestedBundleDigest,
						IsBundle: true,
					},
					{
						Image:    img1DigestRef,
						IsBundle: false,
					},
					{
						Image:    simpleBundle + simpleBundleDigest,
						IsBundle: true,
					},
				},
			})
		})

		logger.Section("download the locations file for nested bundle and check it", func() {
			downloadAndCheckLocationsFile(t, env, nestedBundleDigest[1:], bundle.ImageLocationsConfig{
				APIVersion: "imgpkg.carvel.dev/v1alpha1",
				Kind:       "ImageLocations",
				Images: []bundle.ImageLocation{
					{
						Image:    img1DigestRef,
						IsBundle: false,
					},
					{
						Image:    img2DigestRef,
						IsBundle: false,
					},
					{
						Image:    simpleBundle + simpleBundleDigest,
						IsBundle: true,
					},
				},
			})
		})

		logger.Section("download the locations file for simple bundle and check it", func() {
			downloadAndCheckLocationsFile(t, env, simpleBundleDigest[1:], bundle.ImageLocationsConfig{
				APIVersion: "imgpkg.carvel.dev/v1alpha1",
				Kind:       "ImageLocations",
				Images: []bundle.ImageLocation{
					{
						Image:    img1DigestRef,
						IsBundle: false,
					},
				},
			})
		})
	})
}

func downloadAndCheckLocationsFile(t *testing.T, env *helpers.Env, bundleDigest string, expectedLocation bundle.ImageLocationsConfig) {
	hash, err := v1.NewHash(bundleDigest)
	require.NoError(t, err)
	locationImg := fmt.Sprintf("%s:%s-%s.image-locations.imgpkg", env.RelocationRepo, hash.Algorithm, hash.Hex)

	locationImgFolder := env.Assets.CreateTempFolder("locations-img")
	env.ImageFactory.Download(locationImg, locationImgFolder)

	locationsFilePath := filepath.Join(locationImgFolder, "image-locations.yml")
	require.FileExists(t, locationsFilePath)

	cfg, err := bundle.NewLocationConfigFromPath(locationsFilePath)
	require.NoError(t, err)

	assert.Equal(t, expectedLocation, cfg)
}

func TestCopyBundleUsingTar(t *testing.T) {
	logger := helpers.Logger{}
	t.Run("when a bundle contains other bundles it copies all images from all bundles", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("nested-bundles-tar-test")
		tarFilePath := filepath.Join(testDir, "bundle.tar")

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest string
		logger.Section("create 2 simple images", func() {
			img1DigestRef = imgRef.Context().Name() + "-img1"
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img1DigestRef)
			img1DigestRef = img1DigestRef + img1Digest

			img2DigestRef = imgRef.Context().Name() + "-img2"
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img2DigestRef)
			img2DigestRef = img2DigestRef + img2Digest
		})

		nestedBundle := imgRef.Context().Name() + "-bundle-nested"
		nestedBundleDigest := ""
		logger.Section("create nested bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
`, img1DigestRef, img2DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", nestedBundle, "-f", bundleDir})
			nestedBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		outerBundle := imgRef.Context().Name() + "-bundle-outer"
		outerBundleDigest := ""
		logger.Section("create outer bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
`, nestedBundle+nestedBundleDigest, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", outerBundle, "-f", bundleDir})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("export full bundle to tar", func() {
			imgpkg.Run([]string{"copy", "-b", outerBundle + outerBundleDigest, "--to-tar", tarFilePath})
		})

		lockFilePath := filepath.Join(testDir, "relocate-from-tar-lock.yml")
		logger.Section("import bundle to new repository", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})
			relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, outerBundleDigest)
			env.Assert.AssertBundleLock(lockFilePath, relocatedRef, "")
		})

		imagesToCheck := []string{
			env.RelocationRepo + outerBundleDigest,
			env.RelocationRepo + nestedBundleDigest,
			env.RelocationRepo + img1Digest,
			env.RelocationRepo + img2Digest,
		}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(imagesToCheck))
	})

	t.Run("With Bundle with Nested Bundles, When Push, Copy to Repo, Copy to Tar, Copy to Repo1 and Copy to tar, should copy all Images", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()
		fakeRegBuilder := helpers.NewFakeRegistry(t, env.Logger)
		testDir := env.Assets.CreateTempFolder("nested-bundles-tar-test")
		tarFilePath := filepath.Join(testDir, "bundle.tar")
		secondTarFilePath := filepath.Join(testDir, "second-bundle.tar")

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)
		imgRefFakeReg, err := regname.ParseReference(fakeRegBuilder.ReferenceOnTestServer(imgRef.Identifier()))
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest string
		logger.Section("create 2 simple images", func() {
			img1DigestRef = imgRefFakeReg.Context().Name() + "-img1"
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img1DigestRef)
			img1DigestRef = img1DigestRef + img1Digest

			img2DigestRef = imgRefFakeReg.Context().Name() + "-img2"
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img2DigestRef)
			img2DigestRef = img2DigestRef + img2Digest
		})

		nestedBundle := imgRefFakeReg.Context().Name() + "-bundle-nested"
		nestedBundleDigest := ""
		logger.Section("create nested bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
`, img1DigestRef, img2DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", nestedBundle, "-f", bundleDir})
			nestedBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		outerBundle := imgRef.Context().Name() + "-bundle-outer"
		outerBundleDigest := ""
		logger.Section("create outer bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
- image: %s
`, nestedBundle+nestedBundleDigest, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", outerBundle, "-f", bundleDir})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})
		relocatedOuterBundle := outerBundle + "-relocated"
		logger.Section("relocate OuterBundle", func() {
			imgpkg.Run([]string{"copy", "-b", outerBundle + outerBundleDigest, "--to-repo", relocatedOuterBundle})
		})

		var outputTarExport string
		logger.Section("export full bundle to tar", func() {
			outputTarExport = imgpkg.Run([]string{"copy", "-b", relocatedOuterBundle + outerBundleDigest, "--to-tar", tarFilePath})
		})

		logger.Section("import bundle to new repository", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo})
		})

		var secondOutputTarExport string
		logger.Section("export again to a tar", func() {
			secondOutputTarExport = imgpkg.Run([]string{"copy", "-b", env.RelocationRepo + outerBundleDigest, "--to-tar", secondTarFilePath})
		})

		require.Contains(t, outputTarExport, "exporting 4 images...")
		require.Contains(t, secondOutputTarExport, "exporting 4 images...")
	})

	t.Run("when bundle contains only images it copies all images", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("tar-test")
		tarFilePath := filepath.Join(testDir, "bundle.tar")

		var bundleDigest, imageDigest string
		logger.Section("create bundle with image", func() {
			// create generic image
			imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
			imageDigestRef := env.Image + imageDigest

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy images to a tar file", func() {
			imgpkg.Run([]string{"copy", "-b", env.Image, "--to-tar", tarFilePath})
		})

		lockFilePath := filepath.Join(testDir, "relocate-from-tar-lock.yml")

		logger.Section("import images to the new registry", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})
			relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
			env.Assert.AssertBundleLock(lockFilePath, relocatedRef, "latest")
		})

		imagesToCheck := []string{
			env.RelocationRepo + bundleDigest,
			env.RelocationRepo + imageDigest,
		}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(imagesToCheck))
	})

	t.Run("when bundle have tag it preserves the tag in the destination", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("tar-tag-test")
		tarFilePath := filepath.Join(testDir, "bundle.tar")

		var tag int64
		logger.Section("create bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
`)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			tag = time.Now().UnixNano()
			imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s:%v", env.Image, tag), "-f", bundleDir})
		})

		logger.Section("copy images to a tar file", func() {
			imgpkg.Run([]string{"copy", "-b", fmt.Sprintf("%s:%v", env.Image, tag), "--to-tar", tarFilePath})
		})

		logger.Section("import images to the new registry", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo})
		})

		imagesToCheck := []string{
			fmt.Sprintf("%s:%v", env.RelocationRepo, tag),
		}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(imagesToCheck))
	})

	t.Run("when bundle contains signed images it copies all signatures", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("tar-test")
		tarFilePath := filepath.Join(testDir, "bundle.tar")

		var bundleDigest, imageDigest, imageSignatureTag, bundleSignatureTag string
		logger.Section("create bundle with image and signs both", func() {
			imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
			imageDigestRef := env.Image + imageDigest
			imageSignatureTag = env.ImageFactory.SignImage(imageDigestRef)

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			bundleSignatureTag = env.ImageFactory.SignImage(env.Image + bundleDigest)
		})

		logger.Section("copy images to a tar file", func() {
			imgpkg.Run([]string{"copy", "-b", env.Image, "--to-tar", tarFilePath, "--cosign-signatures"})
		})

		logger.Section("import images to the new registry", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo})
		})

		imagesToCheck := []string{
			env.RelocationRepo + ":" + imageSignatureTag,
			env.RelocationRepo + ":" + bundleSignatureTag,
		}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(imagesToCheck))
		checkSigsOnImages := []string{
			env.RelocationRepo + bundleDigest,
			env.RelocationRepo + imageDigest,
		}
		env.Assert.ValidateCosignSignature(checkSigsOnImages)
	})
}

func TestCopyBundleUsingLockFileAsInput(t *testing.T) {
	logger := helpers.Logger{}
	t.Run("Writing a bundle lockfile uses the 'root' bundle as the reference", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("nested-bundles-tar-test")
		tarFilePath := filepath.Join(testDir, "bundle.tar")

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest string
		logger.Section("create 2 simple images", func() {
			img1DigestRef = imgRef.Context().Name() + "-imgpkg-debug"
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img1DigestRef)
			img1DigestRef = img1DigestRef + img1Digest

			img2DigestRef = imgRef.Context().Name() + "-imgpkg-debug2"
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img2DigestRef)
			img2DigestRef = img2DigestRef + img2Digest
		})

		nestedBundle := imgRef.Context().Name() + "-imgpkg-test"
		nestedBundleDigest := ""
		logger.Section("create nested bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
  annotations:
    kbld.carvel.dev/id: basic-image
- image: %s
  annotations:
    kbld.carvel.dev/id: basic-image
`, img1DigestRef, img2DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", nestedBundle, "-f", bundleDir})
			nestedBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		outerBundle := imgRef.Context().Name() + "-imgpkg-debug-staging"
		outerBundleDigest := ""

		tempLockDirectory := env.Assets.CreateTempFolder("pushed-lock-file")

		logger.Section("create outer bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
  annotations:  
    kbld.carvel.dev/id: basic-bundle
- image: %s
  annotations:
    kbld.carvel.dev/id: basic-image
`, nestedBundle+nestedBundleDigest, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", outerBundle + ":1.0", "-f", bundleDir, "--lock-output", filepath.Join(tempLockDirectory, "staging.lock")})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("export full bundle to tar", func() {
			imgpkg.Run([]string{"copy", "--lock", filepath.Join(tempLockDirectory, "staging.lock"), "--to-tar", tarFilePath})
		})

		lockFilePath := filepath.Join(testDir, "relocate-from-tar-lock.yml")
		logger.Section("import bundle to new repository", func() {
			relocationRepo := env.RelocationRepo + "-imgpkg-debug-release"
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", relocationRepo, "--lock-output", lockFilePath})
			relocatedRef := fmt.Sprintf("%s%s", relocationRepo, outerBundleDigest)

			env.Assert.AssertBundleLock(lockFilePath, relocatedRef, "1.0")
		})
	})
}

func TestCopyErrorsWhenCopyBundleUsingImageFlag(t *testing.T) {
	logger := helpers.Logger{}
	t.Run("when trying to copy a bundle using the -i flag, it fails", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		bundleDigestRef := ""
		logger.Section("create bundle with image", func() {
			imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
			imageDigestRef := env.Image + imageDigest

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			bundleDigestRef = env.Image + bundleDigest
		})

		var stderrBs bytes.Buffer
		_, err := imgpkg.RunWithOpts([]string{"copy", "-i", bundleDigestRef, "--to-tar", "fake_path"},
			helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})
		errOut := stderrBs.String()

		require.Error(t, err)
		assert.Contains(t, errOut, "Expected bundle flag when copying a bundle (hint: Use -b instead of -i for bundles)")
	})
}
