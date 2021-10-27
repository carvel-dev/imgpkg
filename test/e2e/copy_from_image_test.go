// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"

	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
)

func TestCopyImageToRepoDestinationAndOutputImageLockFileAndPreserveImageTag(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	logger := helpers.Logger{}
	tag := time.Now().UnixNano()

	var imageDigest string
	logger.Section(fmt.Sprintf("Create Image with Tag '%d'", tag), func() {
		imageDigest = env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, fmt.Sprintf("%s:%d", env.Image, tag))
	})

	lockOutputPath := filepath.Join(os.TempDir(), "image-relocate-lock.yml")
	defer os.Remove(lockOutputPath)

	logger.Section("Copy Image using the Tag", func() {
		imgpkg.Run([]string{"copy", "--image", fmt.Sprintf("%s:%v", env.Image, tag),
			"--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})
	})

	logger.Section("Check ImagesLock is correct and that Image with copied with tag successfully", func() {
		expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
		env.Assert.AssertImagesLock(lockOutputPath, []lockconfig.ImageRef{{Image: expectedRef}})

		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry([]string{env.RelocationRepo + imageDigest}))

		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry([]string{fmt.Sprintf("%s:%v", env.RelocationRepo, tag)}))
	})

	logger.Section("Check default tag was created", func() {
		algorithmAndSHA := strings.Split(imageDigest, "@")[1]
		splitAlgAndSHA := strings.Split(algorithmAndSHA, ":")
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry([]string{fmt.Sprintf("%s:%s-%s.imgpkg", env.RelocationRepo, splitAlgAndSHA[0], splitAlgAndSHA[1])}))
	})
}

func TestCopyAnImageFromATarToARepoThatDoesNotContainNonDistributableLayersButTheFlagWasIncluded(t *testing.T) {
	t.Run("environment with internet", func(t *testing.T) {
		env := helpers.BuildEnv(t)

		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}

		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath := filepath.Join(testDir, "image.tar")

		nonDistributableLayerDigest := env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo, types.OCIUncompressedRestrictedLayer)

		repoToCopyName := env.RelocationRepo + "include-non-distributable-layers"
		var stdOutWriter bytes.Buffer

		// copy to tar skipping NDL
		imgpkg.Run([]string{"copy", "-i", env.RelocationRepo, "--to-tar", tarFilePath})

		imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName, "--include-non-distributable-layers"}, helpers.RunOpts{
			StderrWriter: &stdOutWriter,
			StdoutWriter: &stdOutWriter,
		})

		digestOfNonDistributableLayer, err := name.NewDigest(repoToCopyName + "@" + nonDistributableLayerDigest)
		require.NoError(t, err)

		layer, err := remote.Layer(digestOfNonDistributableLayer, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		require.NoError(t, err)

		_, err = layer.Compressed()
		require.NoError(t, err)
	})

	t.Run("airgapped environment", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		airgappedRepo, fakeRegistry := startRegistryForAirgapTesting(t, env)

		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}

		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath := filepath.Join(testDir, "image.tar")

		env.ImageFactory.PushImageWithANonDistributableLayer(airgappedRepo, types.OCIUncompressedRestrictedLayer)

		repoToCopyName := env.RelocationRepo + "include-non-distributable-layers"
		var stdOutWriter bytes.Buffer

		// copy to tar skipping NDL
		imgpkg.Run([]string{"copy", "-i", airgappedRepo, "--to-tar", tarFilePath})

		stopRegistryForAirgapTesting(fakeRegistry)

		_, err := imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName, "--include-non-distributable-layers"}, helpers.RunOpts{
			AllowError:   true,
			StderrWriter: &stdOutWriter,
			StdoutWriter: &stdOutWriter,
		})
		require.Error(t, err)

		assert.Regexp(t, "Error: file sha256\\-.*\\.tar\\.gz not found in tar.*(hint: This may be because when copying to a tarball, the --include-non-distributable-layers flag should have been provided.)", stdOutWriter.String())
	})
}

func TestCopyAnImageFromARepoToATarThatDoesNotContainNonDistributableLayersButTheFlagWasIncluded(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()
	logger := helpers.Logger{}

	var tarFilePath string
	logger.Section("Create Image with Non Distributable Layer", func() {
		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath = filepath.Join(testDir, "image.tar")

		env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo, types.OCIUncompressedRestrictedLayer)
	})

	repoToCopyName := env.RelocationRepo + "-include-non-distributable-layers-1"
	var stdOutWriter bytes.Buffer

	logger.Section("copying an image that contains a NDL to a tarball (the tarball includes the NDL)", func() {
		imgpkg.Run([]string{"copy", "-i", env.RelocationRepo, "--to-tar", tarFilePath, "--include-non-distributable-layers"})
	})

	var imageDigest string
	logger.Section("copy from a tarball (with a NDL) to a repo (the image in the repo does *not* include the NDL because the --include-non-dist flag was omitted)", func() {
		stderr := bytes.NewBufferString("")
		imgpkg.RunWithOpts([]string{"copy", "--tty", "--tar", tarFilePath, "--to-repo", repoToCopyName}, helpers.RunOpts{
			StderrWriter: stderr,
		})
		imageDigest = fmt.Sprintf("@%s", env.ImageFactory.ImageDigest(env.RelocationRepo))
	})

	logger.Section("copying from a repo (the image in the repo does *not* include the NDL) to a tarball. We expect NDL to be copied into the tarball", func() {
		imgpkg.Run([]string{"copy", "-i", repoToCopyName + imageDigest, "--to-tar", tarFilePath + "2", "--include-non-distributable-layers"})
		require.NotContains(t, stdOutWriter.String(), "--include-non-distributable-layers")
	})
}

func TestCopyRepoToTarAndThenCopyFromTarToRepo(t *testing.T) {
	logger := helpers.Logger{}
	t.Run("With --include-non-distributable-layers flag and image contains a non-distributable layer should copy every layer", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		repoToCopyName := env.RelocationRepo + "include-non-distributable-layers"
		tarFilePath := ""
		nonDistributableLayerDigest := ""
		logger.Section("Create Image With Non Distributable Layer", func() {
			testDir := env.Assets.CreateTempFolder("image-to-tar")
			tarFilePath = filepath.Join(testDir, "image.tar")

			nonDistributableLayerDigest = env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo, types.OCIUncompressedRestrictedLayer)
		})
		imageDigest := fmt.Sprintf("@%s", env.ImageFactory.ImageDigest(env.RelocationRepo))

		logger.Section("Create a Tar from Image", func() {
			imgpkg.Run([]string{"copy", "-i", env.RelocationRepo, "--to-tar", tarFilePath, "--include-non-distributable-layers"})
		})

		logger.Section("Import Tar to Registry", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName, "--include-non-distributable-layers"})
		})

		logger.Section("Check if Layer was correctly copied", func() {
			digest, err := name.NewDigest(repoToCopyName + "@" + nonDistributableLayerDigest)
			require.NoError(t, err)

			layer, err := remote.Layer(digest, remote.WithAuthFromKeychain(authn.DefaultKeychain))
			require.NoError(t, err)

			_, err = layer.Compressed()
			require.NoError(t, err)
		})

		imgpkg.Run([]string{"pull", "-i", repoToCopyName + imageDigest, "--output", env.Assets.CreateTempFolder("pulled-image")})
	})

	for _, mediaType := range []types.MediaType{types.OCIUncompressedRestrictedLayer, types.DockerForeignLayer} {
		t.Run(fmt.Sprintf("Without --include-non-distributable-layers flag and image contains a non-distributable layer should only copy distributable layers and print a warning message (Using MediaType %s)", mediaType), func(t *testing.T) {
			env := helpers.BuildEnv(t)

			if strings.HasPrefix(env.RelocationRepo, "index.docker.io") {
				t.Skip("Skipping this test due index.docker.io limitation. See https://github.com/docker/hub-feedback/issues/2132")
			}

			if mediaType == types.DockerForeignLayer && strings.HasPrefix(env.RelocationRepo, "ttl.sh") {
				t.Skip("Skipping this test due ttl.sh limitation.")
			}

			if mediaType == types.OCIUncompressedRestrictedLayer && strings.HasPrefix(env.RelocationRepo, "gcr.io") {
				t.Skip("Skipping this test due gcr.io limitation.")
			}

			airgappedRepo, fakeRegistry := startRegistryForAirgapTesting(t, env)

			imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
			defer env.Cleanup()

			repoToCopyName := env.RelocationRepo + "include-non-distributable-layers"

			var tarFilePath, nonDistributableLayerDigest string
			logger.Section("Create Image With Non Distributable Layer", func() {
				testDir := env.Assets.CreateTempFolder("image-to-tar")
				tarFilePath = filepath.Join(testDir, "image.tar")

				nonDistributableLayerDigest = env.ImageFactory.PushImageWithANonDistributableLayer(airgappedRepo, mediaType)
			})

			logger.Section("Create tar from Image", func() {
				imgpkg.Run([]string{"copy", "-i", airgappedRepo, "--to-tar", tarFilePath, "--include-non-distributable-layers"})
			})

			stopRegistryForAirgapTesting(fakeRegistry)

			var stdOutWriter bytes.Buffer
			imgpkg.RunWithOpts([]string{"copy", "--tty", "--tar", tarFilePath, "--to-repo", repoToCopyName}, helpers.RunOpts{
				StdoutWriter: &stdOutWriter,
				StderrWriter: &stdOutWriter,
			})

			logger.Section("Check that non distributable layer was not copied", func() {
				require.Contains(t, stdOutWriter.String(), "Skipped layer due to it being non-distributable.")

				digestOfNonDistributableLayer, err := name.NewDigest(repoToCopyName + "@" + nonDistributableLayerDigest)
				require.NoError(t, err)

				layer, err := remote.Layer(digestOfNonDistributableLayer, remote.WithAuthFromKeychain(authn.DefaultKeychain))
				require.NoError(t, err)

				_, err = layer.Compressed()
				require.Error(t, err)
			})
		})
	}

	t.Run("With --lock-output flag should generate a valid ImageLock file", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		var tagRef, imageDigest, tarFilePath, testDir string
		logger.Section("Create Image with a specific tag", func() {
			testDir = env.Assets.CreateTempFolder("image-to-tar")
			tarFilePath = filepath.Join(testDir, "image.tar")

			tag := fmt.Sprintf("%d", time.Now().UnixNano())
			tagRef = fmt.Sprintf("%s:%s", env.Image, tag)
			out := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, tagRef)
			imageDigest = fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
		})

		logger.Section("Create tar from Image", func() {
			imgpkg.Run([]string{"copy", "-i", tagRef, "--to-tar", tarFilePath})
		})

		lockOutputPath := filepath.Join(testDir, "relocate-from-tar-lock.yml")
		logger.Section("Import Tar into Registry and regenerate a lock file", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})
		})

		logger.Section("Check that Image was correctly imported and ImagesLock is correct", func() {
			expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
			env.Assert.AssertImagesLock(lockOutputPath, []lockconfig.ImageRef{{Image: expectedRef}})

			refs := []string{env.RelocationRepo + imageDigest}
			require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
		})
	})

	t.Run("Preserves tag", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		var tarFilePath, testDir string
		tag := fmt.Sprintf("%d", time.Now().UnixNano())
		tagRef := fmt.Sprintf("%s:%s", env.Image, tag)

		logger.Section("Create Image with a specific tag", func() {
			testDir = env.Assets.CreateTempFolder("image-to-tar")
			tarFilePath = filepath.Join(testDir, "image.tar")

			env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, tagRef)
		})

		logger.Section("Create tar from Image", func() {
			imgpkg.Run([]string{"copy", "-i", tagRef, "--to-tar", tarFilePath})
		})

		logger.Section("Import Tar into Registry and regenerate a lock file", func() {
			imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo})
		})

		logger.Section("Check that the tag is present", func() {
			refs := []string{fmt.Sprintf("%s:%s", env.RelocationRepo, tag)}
			require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
		})
	})

	t.Run("Copies signature", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		var tarFilePath, testDir, signatureTag string
		tag := fmt.Sprintf("%d", time.Now().UnixNano())
		tagRef := fmt.Sprintf("%s:%s", env.Image, tag)

		logger.Section("Create Image and signature", func() {
			testDir = env.Assets.CreateTempFolder("image-to-tar")
			tarFilePath = filepath.Join(testDir, "image.tar")

			imgDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, tagRef)
			imgRef := env.Image + imgDigest
			signatureTag = env.ImageFactory.SignImage(imgRef)
		})

		logger.Section("Create tar from Image", func() {
			imgpkg.Run([]string{"copy",
				"-i", tagRef,
				"--to-tar", tarFilePath,
				"--cosign-signatures",
			})
		})

		logger.Section("Import Tar into Registry and regenerate a lock file", func() {
			imgpkg.Run([]string{"copy",
				"--tar", tarFilePath,
				"--to-repo", env.RelocationRepo,
			})
		})

		logger.Section("Check that signature image is present", func() {
			refs := []string{fmt.Sprintf("%s:%s", env.RelocationRepo, signatureTag)}
			require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
			env.Assert.ValidateCosignSignature([]string{fmt.Sprintf("%s:%s", env.RelocationRepo, tag)})
		})
	})
}

func TestCopyErrors(t *testing.T) {
	logger := helpers.Logger{}
	t.Run("When copying an Image using the -b flag", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		var imageDigestRef string
		logger.Section("Create Images", func() {
			out := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
			imageDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
			imageDigestRef = env.Image + imageDigest
		})

		var stderrBs bytes.Buffer
		_, err := imgpkg.RunWithOpts([]string{"copy", "-b", imageDigestRef, "--to-tar", "fake_path"},
			helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})
		errOut := stderrBs.String()

		require.Error(t, err)
		assert.Contains(t, errOut, "Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
	})

	t.Run("When Copy to Tar while trying to generate a Lock File", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		_, err := imgpkg.RunWithOpts(
			[]string{"copy", "--tty", "-i", env.Image, "--to-tar", "file", "--lock-output", "bogus"},
			helpers.RunOpts{AllowError: true},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "output lock file with tar destination")
	})
}

func stopRegistryForAirgapTesting(fakeRegistry *helpers.FakeTestRegistryBuilder) {
	fakeRegistry.CleanUp()
}

func startRegistryForAirgapTesting(t *testing.T, env *helpers.Env) (string, *helpers.FakeTestRegistryBuilder) {
	fakeRegistry := helpers.NewFakeRegistry(t, env.Logger)

	env.AddCleanup(func() {
		fakeRegistry.CleanUp()
	})

	return fakeRegistry.ReferenceOnTestServer("repo/airgapped-image"), fakeRegistry
}
