// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"carvel.dev/imgpkg/test/helpers"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescribe_TextOutput(t *testing.T) {
	logger := &helpers.Logger{}

	t.Run("bundle with a single image", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
		var bundleDigest, imageDigest, imgSigTag, bundleSigTag, imgSigDigest, bundleSigDigest string
		logger.Section("create bundle with image", func() {
			imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
  annotations:
    some.other.annotation: some other value
    some.annotation: some value
`, env.Image, imageDigest)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s%s", env.Image, bundleTag), "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))

			logger.Section("sign image and Bundle", func() {
				imgSigTag = env.ImageFactory.SignImage(fmt.Sprintf("%s%s", env.Image, imageDigest))
				imgSigDigest = env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s", env.Image, imgSigTag))
				bundleSigTag = env.ImageFactory.SignImage(fmt.Sprintf("%s%s", env.Image, bundleTag))
				bundleSigDigest = env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s", env.Image, bundleSigTag))
			})
		})

		locationsImgDigest := ""
		logger.Section("copy bundle to repository", func() {
			imgpkg.Run([]string{"copy",
				"--bundle", fmt.Sprintf("%s%s", env.Image, bundleDigest),
				"--to-repo", env.RelocationRepo,
				"--cosign-signatures",
			},
			)
			locationsImgDigest = env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s.image-locations.imgpkg", env.RelocationRepo, strings.ReplaceAll(bundleDigest[1:], ":", "-")))
		})

		logger.Section("executes describe command", func() {
			stdout := imgpkg.Run(
				[]string{"describe",
					"--bundle", fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest),
				},
			)
			fmt.Printf("\n\nOutput: %s\n\n", stdout)
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s%s
    Type: Image
    Origin: %s%s
	Layers:
	- Digest: "sha256:a37f35f3e418ea6c1b339df0fc89c8d3155d937740445906ba71466996fac625"
    Annotations:
      some.annotation: some value
      some.other.annotation: some other value
`, env.RelocationRepo, imageDigest, env.Image, imageDigest))

			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s@%s
    Type: Signature
	Layers:
	  - Digest: "sha256:4197c5ac7fb0ba1e597a2377a1b58332d10f6de53dce9c89fd96a3f37034f88b"
    Annotations:
      tag: %s
`, env.RelocationRepo, imgSigDigest, imgSigTag))
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s@%s
    Type: Signature
	Layers:
	  - Digest: "sha256:28014a7e4bef0b7c3f1da47d095f8bb131f474bd5fc96caa4d6125818220b00e"
    Annotations:
      tag: %s
`, env.RelocationRepo, bundleSigDigest, bundleSigTag))

			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s@%s
    Type: Internal
	Layers:
	  - 
	  Digest: "sha256:5d43e9fc8f1ad1b339b5f37bb050b150640ad2c1594345178f1fb38656583a94"
`, env.RelocationRepo, locationsImgDigest))
		})
	})

	t.Run("bundle with bundle collocated", func(t *testing.T) {
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
  annotations:
    what is this: this is the nested bundle
- image: %s
  annotations:
    what is this: this is just an image
`, nestedBundle+nestedBundleDigest, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", outerBundle, "-f", bundleDir})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to a different repository", func() {
			imgpkg.Run([]string{"copy", "-b", outerBundle + outerBundleDigest, "--to-repo", env.RelocationRepo})
		})

		logger.Section("executes describe command", func() {
			stdout := imgpkg.Run(
				[]string{"describe",
					"--bundle", fmt.Sprintf("%s%s", env.RelocationRepo, outerBundleDigest),
				},
			)

			locationsNestedBundleImgDigest := env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s.image-locations.imgpkg", env.RelocationRepo, strings.ReplaceAll(nestedBundleDigest[1:], ":", "-")))
			locationsOuterBundleImgDigest := env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s.image-locations.imgpkg", env.RelocationRepo, strings.ReplaceAll(outerBundleDigest[1:], ":", "-")))

			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s%s
    Type: Bundle
    Origin: %s%s
    Annotations:
      what is this: this is the nested bundle
`, env.RelocationRepo, nestedBundleDigest, nestedBundle, nestedBundleDigest))

			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s%s
      Type: Image
      Origin: %s
`, env.RelocationRepo, img1Digest, img1DigestRef))
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s%s
      Type: Image
      Origin: %s
`, env.RelocationRepo, img2Digest, img2DigestRef))
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s@%s
      Type: Internal
`, env.RelocationRepo, locationsNestedBundleImgDigest))
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s%s
    Type: Image
    Origin: %s
    Annotations:
      what is this: this is just an image
`, env.RelocationRepo, img1Digest, img1DigestRef))
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s@%s
    Type: Internal
`, env.RelocationRepo, locationsOuterBundleImgDigest))
		})
	})

	t.Run("bundle with bundle NOT collocated", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest, img2SigDigest, img2SigTag string
		logger.Section("create 2 simple images", func() {
			img1DigestRef = imgRef.Context().Name() + "-img1"
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img1DigestRef)
			img1DigestRef = img1DigestRef + img1Digest

			img2DigestRef = imgRef.Context().Name() + "-img2"
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img2DigestRef)
			img2DigestRef = img2DigestRef + img2Digest

			img2SigTag = env.ImageFactory.SignImage(img2DigestRef)
			img2SigDigest = env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s", imgRef.Context().Name()+"-img2", img2SigTag))
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
`, img2DigestRef, img1DigestRef)

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

		logger.Section("executes describe command", func() {
			stdout := imgpkg.Run(
				[]string{"describe",
					"--bundle", fmt.Sprintf("%s%s", outerBundle, outerBundleDigest),
					"-o", "text",
				},
			)

			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s%s
    Type: Bundle
    Origin: %s%s
`, nestedBundle, nestedBundleDigest, nestedBundle, nestedBundleDigest))
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s
      Type: Image
      Origin: %s
`, img1DigestRef, img1DigestRef))
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s
      Type: Image
      Origin: %s
`, img2DigestRef, img2DigestRef))
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s@%s
      Type: Signature
      Annotations:
        tag: %s
`, imgRef.Context().Name()+"-img2", img2SigDigest, img2SigTag))

			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s
      Type: Image
      Origin: %s
`, img1DigestRef, img1DigestRef))
		})
	})
}

func TestDescribe_YAMLOutput(t *testing.T) {
	logger := &helpers.Logger{}

	t.Run("bundle with a single image", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
		var bundleDigest, imageDigest, imgSigTag, bundleSigTag, imgSigDigest, bundleSigDigest string
		logger.Section("create bundle with image", func() {
			imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
  annotations:
    some.other.annotation: some other value
    some.annotation: some value
`, env.Image, imageDigest)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s%s", env.Image, bundleTag), "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))

			logger.Section("sign image and Bundle", func() {
				imgSigTag = env.ImageFactory.SignImage(fmt.Sprintf("%s%s", env.Image, imageDigest))
				imgSigDigest = env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s", env.Image, imgSigTag))
				bundleSigTag = env.ImageFactory.SignImage(fmt.Sprintf("%s%s", env.Image, bundleTag))
				bundleSigDigest = env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s", env.Image, bundleSigTag))
			})
		})

		logger.Section("copy bundle to repository", func() {
			imgpkg.Run([]string{"copy",
				"--bundle", fmt.Sprintf("%s%s", env.Image, bundleDigest),
				"--to-repo", env.RelocationRepo,
				"--cosign-signatures",
			},
			)
		})

		logger.Section("executes describe command", func() {
			stdout := imgpkg.Run(
				[]string{"describe",
					"--bundle", fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest),
					"-o", "yaml",
				},
			)
			locationsImgDigest := env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s.image-locations.imgpkg", env.RelocationRepo, strings.ReplaceAll(bundleDigest[1:], ":", "-")))

			stdoutLines := strings.Split(stdout, "\n")
			stdout = strings.Join(stdoutLines[:len(stdoutLines)-1], "\n")
			require.YAMLEq(t, fmt.Sprintf(`sha: %s
content:
  images:
    "%s":
      annotations:
        some.annotation: some value
        some.other.annotation: some other value
      image: %s%s
      imageType: Image
      origin: %s%s
    "%s":
      annotations:
        tag: %s
      image: %s@%s
      imageType: Signature
      origin: %s@%s
    "%s":
      annotations:
        tag: %s
      image: %s@%s
      imageType: Signature
      origin: %s@%s
    "%s":
      image: %s@%s
      imageType: Internal
      origin: %s@%s
image: %s%s
metadata: {}
origin: %s%s
`, bundleDigest[1:],
				imageDigest[1:],
				env.RelocationRepo, imageDigest,
				env.Image, imageDigest,
				bundleSigDigest,
				bundleSigTag,
				env.RelocationRepo, bundleSigDigest,
				env.RelocationRepo, bundleSigDigest,
				imgSigDigest,
				imgSigTag,
				env.RelocationRepo, imgSigDigest,
				env.RelocationRepo, imgSigDigest,
				locationsImgDigest,
				env.RelocationRepo, locationsImgDigest,
				env.RelocationRepo, locationsImgDigest,
				env.RelocationRepo, bundleDigest, env.RelocationRepo, bundleDigest), stdout)
		})
	})

	t.Run("bundle prints output even when tty=false", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
		var bundleDigest, imageDigest, imgSigTag, bundleSigTag, imgSigDigest, bundleSigDigest string
		logger.Section("create bundle with image", func() {
			imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
  annotations:
    some.other.annotation: some other value
    some.annotation: some value
`, env.Image, imageDigest)
			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s%s", env.Image, bundleTag), "-f", bundleDir})
			bundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))

			logger.Section("sign image and Bundle", func() {
				imgSigTag = env.ImageFactory.SignImage(fmt.Sprintf("%s%s", env.Image, imageDigest))
				imgSigDigest = env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s", env.Image, imgSigTag))
				bundleSigTag = env.ImageFactory.SignImage(fmt.Sprintf("%s%s", env.Image, bundleTag))
				bundleSigDigest = env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s", env.Image, bundleSigTag))
			})
		})

		logger.Section("copy bundle to repository", func() {
			imgpkg.Run([]string{"copy",
				"--bundle", fmt.Sprintf("%s%s", env.Image, bundleDigest),
				"--to-repo", env.RelocationRepo,
				"--cosign-signatures",
			},
			)
		})

		logger.Section("executes describe command", func() {
			stdout := imgpkg.Run(
				[]string{"describe",
					"--tty=false", "--bundle", fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest),
					"-o", "yaml",
				},
			)
			locationsImgDigest := env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s.image-locations.imgpkg", env.RelocationRepo, strings.ReplaceAll(bundleDigest[1:], ":", "-")))

			require.YAMLEq(t, fmt.Sprintf(`content:
  images:
    "%s":
      annotations:
        some.annotation: some value
        some.other.annotation: some other value
      image: %s%s
      imageType: Image
      origin: %s%s
    "%s":
      annotations:
        tag: %s
      image: %s@%s
      imageType: Signature
      origin: %s@%s
    "%s":
      annotations:
        tag: %s
      image: %s@%s
      imageType: Signature
      origin: %s@%s
    "%s":
      image: %s@%s
      imageType: Internal
      origin: %s@%s
metadata: {}
image: %s%s
origin: %s%s
sha: %s
`,
				imageDigest[1:],
				env.RelocationRepo, imageDigest,
				env.Image, imageDigest,
				bundleSigDigest,
				bundleSigTag,
				env.RelocationRepo, bundleSigDigest,
				env.RelocationRepo, bundleSigDigest,
				imgSigDigest,
				imgSigTag,
				env.RelocationRepo, imgSigDigest,
				env.RelocationRepo, imgSigDigest,
				locationsImgDigest,
				env.RelocationRepo, locationsImgDigest,
				env.RelocationRepo, locationsImgDigest,
				env.RelocationRepo, bundleDigest, env.RelocationRepo, bundleDigest, bundleDigest[1:]), stdout)
		})
	})

	t.Run("bundle with bundle collocated with yaml output", func(t *testing.T) {
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
  annotations:
    what is this: this is the nested bundle
- image: %s
  annotations:
    what is this: this is just an image
`, nestedBundle+nestedBundleDigest, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out := imgpkg.Run([]string{"push", "--tty", "-b", outerBundle, "-f", bundleDir})
			outerBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("copy bundle to a different repository", func() {
			imgpkg.Run([]string{"copy", "-b", outerBundle + outerBundleDigest, "--to-repo", env.RelocationRepo})
		})

		logger.Section("executes describe command", func() {
			stdout := imgpkg.Run(
				[]string{"describe",
					"--bundle", fmt.Sprintf("%s%s", env.RelocationRepo, outerBundleDigest),
					"-o", "yaml",
				},
			)

			locationsNestedBundleImgDigest := env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s.image-locations.imgpkg", env.RelocationRepo, strings.ReplaceAll(nestedBundleDigest[1:], ":", "-")))
			locationsOuterBundleImgDigest := env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s.image-locations.imgpkg", env.RelocationRepo, strings.ReplaceAll(outerBundleDigest[1:], ":", "-")))
			stdoutLines := strings.Split(stdout, "\n")
			stdout = strings.Join(stdoutLines[:len(stdoutLines)-1], "\n")
			require.YAMLEq(t, fmt.Sprintf(`sha: %s
content:
  bundles:
    "%s":
      annotations:
        what is this: this is the nested bundle
      content:
        images:
          "%s":
            image: %s%s
            imageType: Image
            origin: %s
          "%s":
            image: %s%s
            imageType: Image
            origin: %s
          "%s":
            image: %s@%s
            imageType: Internal
            origin: %s@%s
      image: %s%s
      metadata: {}
      origin: %s%s
  images:
    "%s":
      annotations:
        what is this: this is just an image
      image: %s%s
      imageType: Image
      origin: %s
    "%s":
      image: %s@%s
      imageType: Internal
      origin: %s@%s
image: %s%s
metadata: {}
origin: %s%s
`,
				outerBundleDigest[1:],
				nestedBundleDigest[1:],
				img1Digest[1:],
				env.RelocationRepo, img1Digest, img1DigestRef,
				img2Digest[1:],
				env.RelocationRepo, img2Digest, img2DigestRef,
				locationsNestedBundleImgDigest,
				env.RelocationRepo, locationsNestedBundleImgDigest, env.RelocationRepo, locationsNestedBundleImgDigest,
				env.RelocationRepo, nestedBundleDigest, nestedBundle, nestedBundleDigest,
				img1Digest[1:],
				env.RelocationRepo, img1Digest, img1DigestRef,
				locationsOuterBundleImgDigest,
				env.RelocationRepo, locationsOuterBundleImgDigest, env.RelocationRepo, locationsOuterBundleImgDigest,
				env.RelocationRepo, outerBundleDigest, env.RelocationRepo, outerBundleDigest,
			), stdout)
		})
	})

	t.Run("bundle with bundle NOT collocated", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		imgRef, err := regname.ParseReference(env.Image)
		require.NoError(t, err)

		var img1DigestRef, img2DigestRef, img1Digest, img2Digest, img2SigDigest, img2SigTag string
		logger.Section("create 2 simple images", func() {
			img1DigestRef = imgRef.Context().Name() + "-img1"
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img1DigestRef)
			img1DigestRef = img1DigestRef + img1Digest

			img2DigestRef = imgRef.Context().Name() + "-img2"
			img2Digest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, img2DigestRef)
			img2DigestRef = img2DigestRef + img2Digest

			img2SigTag = env.ImageFactory.SignImage(img2DigestRef)
			img2SigDigest = env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s", imgRef.Context().Name()+"-img2", img2SigTag))
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
`, img2DigestRef, img1DigestRef)

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

		logger.Section("executes describe command", func() {
			stdout := imgpkg.Run(
				[]string{"describe",
					"--bundle", fmt.Sprintf("%s%s", outerBundle, outerBundleDigest),
					"--output-type", "yaml",
				},
			)

			stdoutLines := strings.Split(stdout, "\n")
			stdout = strings.Join(stdoutLines[:len(stdoutLines)-1], "\n")
			require.YAMLEq(t, fmt.Sprintf(`sha: %s
content:
  bundles:
    "%s":
      content:
        images:
          "%s":
            image: %s
            imageType: Image
            origin: %s
          "%s":
            image: %s
            imageType: Image
            origin: %s
          "%s":
            annotations:
              tag: %s
            image: %s@%s
            imageType: Signature
            origin: %s@%s
      image: %s%s
      metadata: {}
      origin: %s%s
  images:
    "%s":
      image: %s
      imageType: Image
      origin: %s
image: %s%s
metadata: {}
origin: %s%s
`,
				outerBundleDigest[1:],
				nestedBundleDigest[1:],
				img1Digest[1:],
				img1DigestRef, img1DigestRef,
				img2Digest[1:],
				img2DigestRef, img2DigestRef,
				img2SigDigest,
				img2SigTag,
				imgRef.Context().Name()+"-img2", img2SigDigest,
				imgRef.Context().Name()+"-img2", img2SigDigest,
				nestedBundle, nestedBundleDigest, nestedBundle, nestedBundleDigest,
				img1Digest[1:],
				img1DigestRef, img1DigestRef,
				outerBundle, outerBundleDigest, outerBundle, outerBundleDigest,
			), stdout)
		})
	})
}
