// Copyright 2024 The Carvel Authors.
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

			digestSha := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + imageDigest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s%s
    Type: Image
    Origin: %s%s
    Layers:
      - Digest: %s
    Annotations:
      some.annotation: some value
      some.other.annotation: some other value
`, env.RelocationRepo, imageDigest, env.Image, imageDigest, digestSha[0]))

			digestSha = env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + imgSigDigest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s@%s
    Type: Signature
    Layers:
      - Digest: %s
    Annotations:
      tag: %s
`, env.RelocationRepo, imgSigDigest, digestSha[0], imgSigTag))

			digestSha = env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + bundleSigDigest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s@%s
    Type: Signature
    Layers:
      - Digest: %s
    Annotations:
      tag: %s
`, env.RelocationRepo, bundleSigDigest, digestSha[0], bundleSigTag))

			digestSha = env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + locationsImgDigest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s@%s
    Type: Internal
    Layers:
      - Digest: %s
`, env.RelocationRepo, locationsImgDigest, digestSha[0]))
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

			digestSha := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + nestedBundleDigest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s%s
    Type: Bundle
    Origin: %s%s
    Layers:
      - Digest: %s
    Annotations:
      what is this: this is the nested bundle
`, env.RelocationRepo, nestedBundleDigest, nestedBundle, nestedBundleDigest, digestSha[0]))

			digestSha = env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + img1Digest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s%s
      Type: Image
      Origin: %s
      Layers:
        - Digest: %s
`, env.RelocationRepo, img1Digest, img1DigestRef, digestSha[0]))

			digestSha = env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + img2Digest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s%s
      Type: Image
      Origin: %s
      Layers:
        - Digest: %s
`, env.RelocationRepo, img2Digest, img2DigestRef, digestSha[0]))

			digestSha = env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + locationsNestedBundleImgDigest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s@%s
      Type: Internal
      Layers:
        - Digest: %s
`, env.RelocationRepo, locationsNestedBundleImgDigest, digestSha[0]))

			digestSha = env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + img1Digest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s%s
    Type: Image
    Origin: %s
    Layers:
      - Digest: %s
    Annotations:
      what is this: this is just an image
`, env.RelocationRepo, img1Digest, img1DigestRef, digestSha[0]))

			digestSha = env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + locationsOuterBundleImgDigest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s@%s
    Type: Internal
    Layers:
      - Digest: %s
`, env.RelocationRepo, locationsOuterBundleImgDigest, digestSha[0]))
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

			digestSha := env.ImageFactory.GetImageLayersDigest(nestedBundle + nestedBundleDigest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`  - Image: %s%s
    Type: Bundle
    Origin: %s%s
    Layers:
      - Digest: %s
`, nestedBundle, nestedBundleDigest, nestedBundle, nestedBundleDigest, digestSha[0]))

			digestSha = env.ImageFactory.GetImageLayersDigest(img1DigestRef)
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s
      Type: Image
      Origin: %s
      Layers:
        - Digest: %s
`, img1DigestRef, img1DigestRef, digestSha[0]))

			digestSha = env.ImageFactory.GetImageLayersDigest(img2DigestRef)
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s
      Type: Image
      Origin: %s
      Layers:
        - Digest: %s
`, img2DigestRef, img2DigestRef, digestSha[0]))

			digestSha = env.ImageFactory.GetImageLayersDigest(imgRef.Context().Name() + "-img2" + "@" + img2SigDigest)
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s@%s
      Type: Signature
      Layers:
        - Digest: %s
      Annotations:
        tag: %s
`, imgRef.Context().Name()+"-img2", img2SigDigest, digestSha[0], img2SigTag))

			digestSha = env.ImageFactory.GetImageLayersDigest(img1DigestRef)
			assert.Contains(t, stdout, fmt.Sprintf(
				`    - Image: %s
      Type: Image
      Origin: %s
      Layers:
        - Digest: %s
`, img1DigestRef, img1DigestRef, digestSha[0]))
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
			digestSha1 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + imageDigest)
			digestSha2 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + bundleSigDigest)
			digestSha3 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + imgSigDigest)
			digestSha4 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + locationsImgDigest)
			digestSha5 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + bundleDigest)
			require.YAMLEq(t, fmt.Sprintf(`sha: %s
content:
  images:
    "%s":
      annotations:
        some.annotation: some value
        some.other.annotation: some other value
      image: %s%s
      imageType: Image
      layers:
      - digest: %s
      origin: %s%s
    "%s":
      annotations:
        tag: %s
      image: %s@%s
      imageType: Signature
      layers:
      - digest: %s
      origin: %s@%s
    "%s":
      annotations:
        tag: %s
      image: %s@%s
      imageType: Signature
      layers:
      - digest: %s
      origin: %s@%s
    "%s":
      image: %s@%s
      imageType: Internal
      layers:
      - digest: %s
      origin: %s@%s
image: %s%s
layers:
- digest: %s
metadata: {}
origin: %s%s
`, bundleDigest[1:],
				imageDigest[1:],
				env.RelocationRepo, imageDigest,
				digestSha1[0],
				env.Image, imageDigest,
				bundleSigDigest,
				bundleSigTag,
				env.RelocationRepo, bundleSigDigest,
				digestSha2[0],
				env.RelocationRepo, bundleSigDigest,
				imgSigDigest,
				imgSigTag,
				env.RelocationRepo, imgSigDigest,
				digestSha3[0],
				env.RelocationRepo, imgSigDigest,
				locationsImgDigest,
				env.RelocationRepo, locationsImgDigest,
				digestSha4[0],
				env.RelocationRepo, locationsImgDigest,
				env.RelocationRepo, bundleDigest, digestSha5[0], env.RelocationRepo, bundleDigest), stdout)
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
			digestSha1 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + imageDigest)
			digestSha2 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + bundleSigDigest)
			digestSha3 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + imgSigDigest)
			digestSha4 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + locationsImgDigest)
			digestSha5 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + bundleDigest)
			require.YAMLEq(t, fmt.Sprintf(`content:
  images:
    "%s":
      annotations:
        some.annotation: some value
        some.other.annotation: some other value
      image: %s%s
      imageType: Image
      layers:
      - digest: %s
      origin: %s%s
    "%s":
      annotations:
        tag: %s
      image: %s@%s
      imageType: Signature
      layers:
      - digest: %s
      origin: %s@%s
    "%s":
      annotations:
        tag: %s
      image: %s@%s
      imageType: Signature
      layers:
      - digest: %s
      origin: %s@%s
    "%s":
      image: %s@%s
      imageType: Internal
      layers:
      - digest: %s
      origin: %s@%s
metadata: {}
image: %s%s
layers:
- digest: %s
origin: %s%s
sha: %s
`,
				imageDigest[1:],
				env.RelocationRepo, imageDigest, digestSha1[0],
				env.Image, imageDigest,
				bundleSigDigest,
				bundleSigTag,
				env.RelocationRepo, bundleSigDigest, digestSha2[0],
				env.RelocationRepo, bundleSigDigest,
				imgSigDigest,
				imgSigTag,
				env.RelocationRepo, imgSigDigest, digestSha3[0],
				env.RelocationRepo, imgSigDigest,
				locationsImgDigest,
				env.RelocationRepo, locationsImgDigest, digestSha4[0],
				env.RelocationRepo, locationsImgDigest,
				env.RelocationRepo, bundleDigest, digestSha5[0], env.RelocationRepo, bundleDigest, bundleDigest[1:]), stdout)
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
			digestSha1 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + img1Digest)
			digestSha2 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + img2Digest)
			digestSha3 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + locationsNestedBundleImgDigest)
			digestSha4 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + nestedBundleDigest)
			digestSha5 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + img1Digest)
			digestSha6 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + locationsOuterBundleImgDigest)
			digestSha7 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + outerBundleDigest)
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
            layers:
            - digest: %s
            origin: %s
          "%s":
            image: %s%s
            imageType: Image
            layers:
            - digest: %s
            origin: %s
          "%s":
            image: %s@%s
            imageType: Internal
            layers:
            - digest: %s
            origin: %s@%s
      image: %s%s
      layers:
      - digest: %s
      metadata: {}
      origin: %s%s
  images:
    "%s":
      annotations:
        what is this: this is just an image
      image: %s%s
      imageType: Image
      layers:
      - digest: %s
      origin: %s
    "%s":
      image: %s@%s
      imageType: Internal
      layers:
      - digest: %s
      origin: %s@%s
image: %s%s
layers:
- digest: %s
metadata: {}
origin: %s%s
`,
				outerBundleDigest[1:],
				nestedBundleDigest[1:],
				img1Digest[1:],
				env.RelocationRepo, img1Digest, digestSha1[0], img1DigestRef,
				img2Digest[1:],
				env.RelocationRepo, img2Digest, digestSha2[0], img2DigestRef,
				locationsNestedBundleImgDigest,
				env.RelocationRepo, locationsNestedBundleImgDigest, digestSha3[0], env.RelocationRepo, locationsNestedBundleImgDigest,
				env.RelocationRepo, nestedBundleDigest, digestSha4[0], nestedBundle, nestedBundleDigest,
				img1Digest[1:],
				env.RelocationRepo, img1Digest, digestSha5[0], img1DigestRef,
				locationsOuterBundleImgDigest,
				env.RelocationRepo, locationsOuterBundleImgDigest, digestSha6[0], env.RelocationRepo, locationsOuterBundleImgDigest,
				env.RelocationRepo, outerBundleDigest, digestSha7[0], env.RelocationRepo, outerBundleDigest,
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
			digestSha1 := env.ImageFactory.GetImageLayersDigest(img1DigestRef)
			digestSha2 := env.ImageFactory.GetImageLayersDigest(img2DigestRef)
			digestSha3 := env.ImageFactory.GetImageLayersDigest(imgRef.Context().Name() + "-img2" + "@" + img2SigDigest)
			digestSha4 := env.ImageFactory.GetImageLayersDigest(nestedBundle + nestedBundleDigest)
			digestSha5 := env.ImageFactory.GetImageLayersDigest(img1DigestRef)
			digestSha6 := env.ImageFactory.GetImageLayersDigest(outerBundle + outerBundleDigest)
			require.YAMLEq(t, fmt.Sprintf(`sha: %s
content:
  bundles:
    "%s":
      content:
        images:
          "%s":
            image: %s
            imageType: Image
            layers:
            - digest: %s
            origin: %s
          "%s":
            image: %s
            imageType: Image
            layers:
            - digest: %s
            origin: %s
          "%s":
            annotations:
              tag: %s
            image: %s@%s
            imageType: Signature
            layers:
            - digest: %s
            origin: %s@%s
      image: %s%s
      layers:
      - digest: %s
      metadata: {}
      origin: %s%s
  images:
    "%s":
      image: %s
      imageType: Image
      layers:
      - digest: %s
      origin: %s
image: %s%s
layers:
- digest: %s
metadata: {}
origin: %s%s
`,
				outerBundleDigest[1:],
				nestedBundleDigest[1:],
				img1Digest[1:],
				img1DigestRef, digestSha1[0], img1DigestRef,
				img2Digest[1:],
				img2DigestRef, digestSha2[0], img2DigestRef,
				img2SigDigest,
				img2SigTag,
				imgRef.Context().Name()+"-img2", img2SigDigest, digestSha3[0],
				imgRef.Context().Name()+"-img2", img2SigDigest,
				nestedBundle, nestedBundleDigest, digestSha4[0], nestedBundle, nestedBundleDigest,
				img1Digest[1:],
				img1DigestRef, digestSha5[0], img1DigestRef,
				outerBundle, outerBundleDigest, digestSha6[0], outerBundle, outerBundleDigest,
			), stdout)
		})
	})

	t.Run("when describing relocated bundle does not reach to original images", func(t *testing.T) {
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

		var img1DigestRef, img1Digest string
		logger.Section("create 2 simple images", func() {
			img1DigestRef = fakeRegistryBuilder.ReferenceOnTestServer(imgRef.Context().RepositoryStr() + "-img1")
			img1Digest = env.ImageFactory.PushSimpleAppImageWithRandomFileWithAuth(imgpkg, img1DigestRef, fakeRegistryBuilder.Host(), username, password)
			img1DigestRef = img1DigestRef + img1Digest
		})

		privateBundle := fakeRegistryBuilder.ReferenceOnTestServer(imgRef.Context().RepositoryStr() + "-bundle")
		privateBundleDigest := ""
		logger.Section("create bundle", func() {
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, img1DigestRef)

			bundleDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)
			out, err := imgpkg.RunWithOpts([]string{"push", "--tty", "-b", privateBundle, "-f", bundleDir}, helpers.RunOpts{
				EnvVars: []string{
					"IMGPKG_REGISTRY_HOSTNAME=" + fakeRegistryBuilder.Host(),
					"IMGPKG_REGISTRY_USERNAME=" + username,
					"IMGPKG_REGISTRY_PASSWORD=" + password,
				},
			})
			require.NoError(t, err)
			privateBundleDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		publicBundle := env.RelocationRepo
		logger.Section("copy bundle to different registry", func() {
			_, err := imgpkg.RunWithOpts([]string{"copy", "--tty", "-b", privateBundle + privateBundleDigest, "--to-repo", publicBundle}, helpers.RunOpts{
				EnvVars: []string{
					"IMGPKG_REGISTRY_HOSTNAME=" + fakeRegistryBuilder.Host(),
					"IMGPKG_REGISTRY_USERNAME=" + username,
					"IMGPKG_REGISTRY_PASSWORD=" + password,
				},
			})
			require.NoError(t, err)
		})

		logger.Section("executes describe command", func() {
			stdout := imgpkg.Run(
				[]string{"describe",
					"--bundle", fmt.Sprintf("%s%s", env.RelocationRepo, privateBundleDigest),
					"-o", "yaml",
				},
			)

			locationsPublicBundleImgDigest := env.ImageFactory.ImageDigest(fmt.Sprintf("%s:%s.image-locations.imgpkg", env.RelocationRepo, strings.ReplaceAll(privateBundleDigest[1:], ":", "-")))
			stdoutLines := strings.Split(stdout, "\n")
			stdout = strings.Join(stdoutLines[:len(stdoutLines)-1], "\n")
			digestSha1 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + img1Digest)
			digestSha3 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + "@" + locationsPublicBundleImgDigest)
			digestSha4 := env.ImageFactory.GetImageLayersDigest(env.RelocationRepo + privateBundleDigest)
			require.YAMLEq(t, fmt.Sprintf(`sha: %s
content:
  images:
    "%s":
      image: %s%s
      imageType: Image
      layers:
      - digest: %s
      origin: %s
    "%s":
      image: %s@%s
      imageType: Internal
      layers:
      - digest: %s
      origin: %s@%s
image: %s%s
layers:
- digest: %s
metadata: {}
origin: %s%s
`,
				privateBundleDigest[1:], // Bundle SHA

				img1Digest[1:],
				env.RelocationRepo, img1Digest,
				digestSha1[0], // Image 1 Layer digest
				img1DigestRef, // Origin Ref

				locationsPublicBundleImgDigest,
				env.RelocationRepo, locationsPublicBundleImgDigest,
				digestSha3[0], // Locations Image Digest
				env.RelocationRepo, locationsPublicBundleImgDigest,

				env.RelocationRepo, privateBundleDigest, // Bundle Image with digest
				digestSha4[0], // Bundle layer digest
				env.RelocationRepo, privateBundleDigest,
			), stdout)
		})
	})
}
