// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

func TestCopyImageToRepoDestinationAndOutputImageLockFileAndPreserverImageTag(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	// create generic image
	tag := time.Now().UnixNano()
	imageDigest := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, fmt.Sprintf("%s:%d", env.Image, tag))

	lockOutputPath := filepath.Join(os.TempDir(), "image-relocate-lock.yml")
	defer os.Remove(lockOutputPath)

	// copy via create ref
	imgpkg.Run([]string{"copy", "--image", fmt.Sprintf("%s:%v", env.Image, tag),
		"--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	env.Assert.AssertImagesLock(lockOutputPath, []lockconfig.ImageRef{{Image: expectedRef}})

	require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry([]string{env.RelocationRepo + imageDigest}))

	require.Error(t, env.Assert.ValidateImagesPresenceInRegistry([]string{fmt.Sprintf("%s:%v", env.RelocationRepo, tag)}))
}

func TestCopyAnImageFromATarToARepoThatDoesNotContainNonDistributableLayersButTheFlagWasIncluded(t *testing.T) {
	t.Run("environment with internet", func(t *testing.T) {
		env := helpers.BuildEnv(t)

		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}

		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath := filepath.Join(testDir, "image.tar")

		nonDistributableLayerDigest := env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo)

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
		airgappedRepo := startRegistryForAirgapTesting(t, env)

		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}

		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath := filepath.Join(testDir, "image.tar")

		env.ImageFactory.PushImageWithANonDistributableLayer(airgappedRepo)

		repoToCopyName := env.RelocationRepo + "include-non-distributable-layers"
		var stdOutWriter bytes.Buffer

		// copy to tar skipping NDL
		imgpkg.Run([]string{"copy", "-i", airgappedRepo, "--to-tar", tarFilePath})

		stopRegistryForAirgapTesting(t, env)

		_, err := imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName, "--include-non-distributable-layers"}, helpers.RunOpts{
			AllowError:   true,
			StderrWriter: &stdOutWriter,
			StdoutWriter: &stdOutWriter,
		})
		require.Error(t, err)

		assert.Regexp(t, "Error: file sha256\\-.*\\.tar\\.gz not found in tar\\. hint: This may be because when copying to a tarball, the --include-non-distributable-layers flag should have been provided.", stdOutWriter.String())
	})
}

func TestCopyAnImageFromARepoToATarThatDoesNotContainNonDistributableLayersButTheFlagWasIncluded(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	// general setup
	testDir := env.Assets.CreateTempFolder("image-to-tar")
	tarFilePath := filepath.Join(testDir, "image.tar")

	env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo)

	repoToCopyName := env.RelocationRepo + "-include-non-distributable-layers-1"
	var stdOutWriter bytes.Buffer

	// copying an image that contains a NDL to a tarball (the tarball includes the NDL)
	imgpkg.Run([]string{"copy", "-i", env.RelocationRepo, "--to-tar", tarFilePath, "--include-non-distributable-layers"})

	stderr := bytes.NewBufferString("")
	// copy from a tarball (with a NDL) to a repo (the image in the repo does *not* include the NDL because the --include-non-dist flag was omitted)
	imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName}, helpers.RunOpts{
		StderrWriter: stderr,
	})
	imageDigest := fmt.Sprintf("@%s", env.ImageFactory.ImageDigest(env.RelocationRepo))

	// copying from a repo (the image in the repo does *not* include the NDL) to a tarball. We expect NDL to be copied into the tarball.
	imgpkg.Run([]string{"copy", "-i", repoToCopyName + imageDigest, "--to-tar", tarFilePath + "2", "--include-non-distributable-layers"})

	require.NotContains(t, stdOutWriter.String(), "hint: This may be because when copying to a tarball, the --include-non-distributable-layers flag should have been provided")
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

			nonDistributableLayerDigest = env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo)
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

	t.Run("Without --include-non-distributable-layers flag and image contains a non-distributable layer should only copy distributable layers and print a warning message", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		airgappedRepo := startRegistryForAirgapTesting(t, env)

		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		// general setup
		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath := filepath.Join(testDir, "image.tar")

		nonDistributableLayerDigest := env.ImageFactory.PushImageWithANonDistributableLayer(airgappedRepo)
		repoToCopyName := env.RelocationRepo + "include-non-distributable-layers"

		// copy to tar
		imgpkg.Run([]string{"copy", "-i", airgappedRepo, "--to-tar", tarFilePath, "--include-non-distributable-layers"})

		stopRegistryForAirgapTesting(t, env)

		var stdOutWriter bytes.Buffer
		imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName}, helpers.RunOpts{
			StdoutWriter: &stdOutWriter,
			StderrWriter: &stdOutWriter,
		})
		require.Contains(t, stdOutWriter.String(), "Skipped layer due to it being non-distributable.")

		digestOfNonDistributableLayer, err := name.NewDigest(repoToCopyName + "@" + nonDistributableLayerDigest)
		require.NoError(t, err)

		layer, err := remote.Layer(digestOfNonDistributableLayer, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		require.NoError(t, err)

		_, err = layer.Compressed()
		require.Error(t, err)
	})

	t.Run("With --lock-output flag should generate a valid ImageLock file", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
		defer env.Cleanup()

		// general setup
		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath := filepath.Join(testDir, "image.tar")

		// create generic image
		tag := fmt.Sprintf("%d", time.Now().UnixNano())
		tagRef := fmt.Sprintf("%s:%s", env.Image, tag)
		out := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, tagRef)
		imageDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))

		// copy to tar
		imgpkg.Run([]string{"copy", "-i", tagRef, "--to-tar", tarFilePath})

		lockOutputPath := filepath.Join(testDir, "relocate-from-tar-lock.yml")
		defer os.Remove(lockOutputPath)

		// copy from tar to repo
		imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

		expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
		env.Assert.AssertImagesLock(lockOutputPath, []lockconfig.ImageRef{{Image: expectedRef}})

		// check if image is present in dst repo
		refs := []string{env.RelocationRepo + imageDigest}
		require.NoError(t, env.Assert.ValidateImagesPresenceInRegistry(refs))
	})
}

func TestCopyErrorsWhenCopyImageUsingBundleFlag(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	// create generic image
	out := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
	imageDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
	imageDigestRef := env.Image + imageDigest

	var stderrBs bytes.Buffer
	_, err := imgpkg.RunWithOpts([]string{"copy", "-b", imageDigestRef, "--to-tar", "fake_path"},
		helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	require.Error(t, err)
	assert.Contains(t, errOut, "Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
}

func TestCopyErrorsWhenCopyToTarAndGenerateOutputLockFile(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	_, err := imgpkg.RunWithOpts(
		[]string{"copy", "--tty", "-i", env.Image, "--to-tar", "file", "--lock-output", "bogus"},
		helpers.RunOpts{AllowError: true},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "output lock file with tar destination")
}

func stopRegistryForAirgapTesting(t *testing.T, env *helpers.Env) {
	err := exec.Command("docker", "stop", "registry-for-airgapped-testing").Run()
	require.NoError(t, err)

	env.AddCleanup(func() {
		exec.Command("docker", "start", "registry-for-airgapped-testing").Run()
	})
}

func startRegistryForAirgapTesting(t *testing.T, env *helpers.Env) string {
	dockerRunCmd := exec.Command("docker", "run", "-d", "-p", "5000", "--env", "REGISTRY_VALIDATION_MANIFESTS_URLS_ALLOW=- ^https?://", "--restart", "always", "--name", "registry-for-airgapped-testing", "registry:2")
	_, err := dockerRunCmd.CombinedOutput()
	require.NoError(t, err)

	env.AddCleanup(func() {
		exec.Command("docker", "stop", "registry-for-airgapped-testing").Run()
		exec.Command("docker", "rm", "registry-for-airgapped-testing").Run()
	})

	inspectCmd := exec.Command("docker", "inspect", `--format='{{(index (index .NetworkSettings.Ports "5000/tcp") 0).HostPort}}'`, "registry-for-airgapped-testing")
	output, err := inspectCmd.CombinedOutput()
	require.NoError(t, err)

	hostPort := strings.ReplaceAll(string(output), "'", "")
	return fmt.Sprintf("localhost:%s/repo/airgapped-image", strings.ReplaceAll(hostPort, "\n", ""))
}
