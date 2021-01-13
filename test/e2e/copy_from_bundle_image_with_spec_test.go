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

	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

func TestCopyFromBundleImage(t *testing.T) {
	spec.Run(t, "testCopyFromBundleImage", testCopyFromBundleImage)
}

func testCopyFromBundleImage(t *testing.T, when spec.G, it spec.S) {
	when("Copy Bundle Image", func() {
		var (
			env         Env
			imgpkg      Imgpkg
			imageDigest string
		)
		it.Before(func() {
			env = BuildEnv(t)
			imgpkg = Imgpkg{t, Logger{}, env.ImgpkgPath}

			imageDigest = env.ImageFactory.pushSimpleAppImageWithRandomFile(imgpkg, env.Image)
		})

		it.After(func() {
			//env.Assets.cleanCreatedFolders()
		})

		it("preserves the annotations", func() {
			// create generic image
			imageDigestRef := env.Image + imageDigest

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
  annotations:
    greeting: hello world
`, imageDigestRef)
			bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imageLockYAML)

			// create bundle that refs image with --lock-ouput and a random tag based on time
			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest := fmt.Sprintf("@%s", extractDigest(t, out))
			bundleDigestRef := env.Image + bundleDigest

			// copy
			imgpkg.Run([]string{"copy", "-b", bundleDigestRef, "--to-repo", env.RelocationRepo})

			// pull
			testDir := env.Assets.createTempFolder("test-annotation")
			bundleDigestRef = env.RelocationRepo + bundleDigest
			imgpkg.Run([]string{"pull", "-b", bundleDigestRef, "-o", testDir})

			imgLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(testDir, ".imgpkg", "images.yml"))
			require.NoError(t, err)

			greeting, ok := imgLock.Images[0].Annotations["greeting"]
			require.True(t, ok, "could not find annotation greeting in images lock")
			require.Equal(t, "hello world", greeting)
		})

		it("fail to copy when using the -i flag", func() {
			// create generic image
			imageDigestRef := env.Image + imageDigest

			// create a bundle with ref to generic
			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)
			bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest := fmt.Sprintf("@%s", extractDigest(t, out))
			bundleDigestRef := env.Image + bundleDigest

			var stderrBs bytes.Buffer
			_, err := imgpkg.RunWithOpts([]string{"copy", "-i", bundleDigestRef, "--to-tar", "fake_path"},
				RunOpts{AllowError: true, StderrWriter: &stderrBs})
			require.Error(t, err)

			errOut := stderrBs.String()
			require.Contains(t, errOut, "Expected bundle flag when copying a bundle (hint: Use -b instead of -i for bundles)")
		})

		when("images of the bundle are collocated with the bundle", func() {
			it("copies images and creates a BundleLock file when --lock-output is provided", func() {
				// create a bundle with ref to generic
				imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
`, env.Image, imageDigest)
				bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imageLockYAML)

				// create bundle that refs image and a random tag based on time
				bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
				out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s%s", env.Image, bundleTag), "-f", bundleDir})
				bundleDigest := fmt.Sprintf("@%s", extractDigest(t, out))

				lockOutputPath := filepath.Join(env.Assets.createTempFolder("bundle-lock"), "bundle-relocate-lock.yml")
				// copy via created ref
				imgpkg.Run([]string{"copy",
					"--bundle", fmt.Sprintf("%s%s", env.Image, bundleTag),
					"--to-repo", env.RelocationRepo,
					"--lock-output", lockOutputPath},
				)

				expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
				expectedTag := strings.TrimPrefix(bundleTag, ":")
				require.NoError(t, env.BundleFactory.assertBundleLock(lockOutputPath, expectedRef, expectedTag))

				refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleTag, env.RelocationRepo + bundleDigest}
				require.NoError(t, validateImagesPresenceInRegistry(refs), "validating images")
			})
		})

		when("copy to a tar", func() {
			when("copy to a repository", func() {
				var (
					tag          string
					bundleDigest string
					lockFilePath string
				)
				it.Before(func() {
					testFolder := env.Assets.createTempFolder("tar-folder")
					tarFilePath := filepath.Join(testFolder, "bundle.tar")

					imageDigestRef := env.Image + imageDigest
					imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

					// create a bundle with ref to generic
					bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imageLockYAML)

					tag = fmt.Sprintf("%v", time.Now().UnixNano())
					// create bundle that refs image
					out := imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s:%v", env.Image, tag), "-f", bundleDir})
					bundleDigest = fmt.Sprintf("@%s", extractDigest(t, out))

					// copy to a tar
					imgpkg.Run([]string{"copy", "-b", fmt.Sprintf("%s:%v", env.Image, tag), "--to-tar", tarFilePath})

					lockFilePath = filepath.Join(testFolder, "relocate-from-tar-lock.yml")

					// copy from tar to repo
					imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})
				})

				it("maintain tag of the original bundle", func() {
					relocatedBundleRef := env.RelocationRepo + bundleDigest
					relocatedImageRef := env.RelocationRepo + imageDigest
					relocatedBundleTagRef := fmt.Sprintf("%s:%v", env.RelocationRepo, tag)

					err := validateImagesPresenceInRegistry([]string{relocatedBundleRef, relocatedImageRef, relocatedBundleTagRef})
					require.NoError(t, err)
				})

				it("generates the BundleLock file", func() {
					relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
					require.NoError(t, env.BundleFactory.assertBundleLock(lockFilePath, relocatedRef, fmt.Sprintf("%v", tag)))
				})
			})
		})
	})

	when("Copy Bundle Image with none collocated images", func() {
		it("is successful", func() {
			env := BuildEnv(t)
			imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

			image := env.Image + "-image-outside-repo"
			imageDigest := env.ImageFactory.pushSimpleAppImageWithRandomFile(imgpkg, image)
			// image intentionally does not exist in bundle repo
			imageDigestRef := image + imageDigest

			imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)
			bundleDir := env.BundleFactory.createBundleDir(bundleYAML, imageLockYAML)

			out := imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", bundleDir})
			bundleDigest := fmt.Sprintf("@%s", extractDigest(t, out))
			bundleDigestRef := env.Image + bundleDigest

			imgpkg.Run([]string{"copy", "--bundle", bundleDigestRef, "--to-repo", env.RelocationRepo})

			refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest}
			require.NoError(t, validateImagesPresenceInRegistry(refs), "validating image presence")
		})
	})
}
