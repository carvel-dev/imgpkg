// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/test/helpers"

	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

func TestCopyImageToRepoDestinationAndOutputImageLockFileAndPreserverImageTag(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
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

	if err := env.Assert.ValidateImagesPresenceInRegistry([]string{env.RelocationRepo + imageDigest}); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}

	if err := env.Assert.ValidateImagesPresenceInRegistry([]string{fmt.Sprintf("%s:%v", env.RelocationRepo, tag)}); err == nil {
		t.Fatalf("expected not to find image with tag '%v', but did", tag)
	}
}

func TestCopyAnImageFromATarToARepoThatDoesNotContainNonDistributableLayersButTheFlagWasIncluded(t *testing.T) {
	t.Run("environment with internet", func(t *testing.T) {
		env := helpers.BuildEnv(t)

		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}

		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath := filepath.Join(testDir, "image.tar")

		nonDistributableLayerDigest := env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo)

		repoToCopyName := env.RelocationRepo + "include-non-distributable"
		var stdOutWriter bytes.Buffer

		// copy to tar skipping NDL
		imgpkg.Run([]string{"copy", "-i", env.RelocationRepo, "--to-tar", tarFilePath})

		imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName, "--include-non-distributable"}, helpers.RunOpts{
			StderrWriter: &stdOutWriter,
			StdoutWriter: &stdOutWriter,
		})

		digestOfNonDistributableLayer, err := name.NewDigest(repoToCopyName + "@" + nonDistributableLayerDigest)
		if err != nil {
			t.Fatalf("Unable to determine the digest of the non-distributable layer. Got: %v", err)
		}

		layer, err := remote.Layer(digestOfNonDistributableLayer, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if err != nil {
			t.Fatalf("Unable to fetch the layer of the copied image: %v", err)
		}

		_, err = layer.Compressed()
		if err != nil {
			t.Fatalf("Expected non-distributable layer to be copied into registry, however it wasn't")
		}
	})

	t.Run("airgapped environment", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		airgappedRepo := startRegistryForAirgapTesting(t, env)

		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}

		defer env.Cleanup()

		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath := filepath.Join(testDir, "image.tar")

		env.ImageFactory.PushImageWithANonDistributableLayer(airgappedRepo)

		repoToCopyName := env.RelocationRepo + "include-non-distributable"
		var stdOutWriter bytes.Buffer

		// copy to tar skipping NDL
		imgpkg.Run([]string{"copy", "-i", airgappedRepo, "--to-tar", tarFilePath})

		stopRegistryForAirgapTesting(t, env)

		_, err := imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName, "--include-non-distributable"}, helpers.RunOpts{
			AllowError:   true,
			StderrWriter: &stdOutWriter,
			StdoutWriter: &stdOutWriter,
		})

		if err == nil {
			t.Fatalf("Expected copy command to fail but it did not")
		}

		if !regexp.MustCompile("Error: file sha256\\-.*\\.tar\\.gz not found in tar\\. hint: This may be because when copying to a tarball, the --include-non-distributable flag should have been provided.").MatchString(stdOutWriter.String()) {
			t.Fatalf("Expected warning message to user, specifying tarball did not contain a non-distributable layer. But got: %s", stdOutWriter.String())
		}
	})
}

func TestCopyAnImageFromARepoToATarThatDoesNotContainNonDistributableLayersButTheFlagWasIncluded(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	// general setup
	testDir := env.Assets.CreateTempFolder("image-to-tar")
	tarFilePath := filepath.Join(testDir, "image.tar")

	env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo)

	repoToCopyName := env.RelocationRepo + "include-non-distributable-1"
	var stdOutWriter bytes.Buffer

	// copying an image that contains a NDL to a tarball (the tarball includes the NDL)
	imgpkg.Run([]string{"copy", "-i", env.RelocationRepo, "--to-tar", tarFilePath, "--include-non-distributable"})

	stderr := bytes.NewBufferString("")
	// copy from a tarball (with a NDL) to a repo (the image in the repo does *not* include the NDL because the --include-non-dist flag was omitted)
	imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName}, helpers.RunOpts{
		StderrWriter: stderr,
	})
	imageDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, stderr.String()))

	// copying from a repo (the image in the repo does *not* include the NDL) to a tarball. We expect NDL to be copied into the tarball.
	imgpkg.Run([]string{"copy", "-i", repoToCopyName + imageDigest, "--to-tar", tarFilePath + "2", "--include-non-distributable"})

	if strings.Contains(stdOutWriter.String(), "hint: This may be because when copying to a tarball, the --include-non-distributable flag should have been provided") {
		t.Fatalf("Expected no warning message. But got: %s", stdOutWriter.String())
	}
}

func TestCopyRepoToTarAndThenCopyFromTarToRepo(t *testing.T) {
	t.Run("With --include-non-distributable flag and image contains a non-distributable layer should copy every layer", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()

		// general setup
		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath := filepath.Join(testDir, "image.tar")

		nonDistributableLayerDigest := env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo)
		repoToCopyName := env.RelocationRepo + "include-non-distributable"

		// copy to tar
		imgpkg.Run([]string{"copy", "-i", env.RelocationRepo, "--to-tar", tarFilePath, "--include-non-distributable"})

		stderr := bytes.NewBufferString("")
		imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName, "--include-non-distributable"}, helpers.RunOpts{
			StderrWriter: stderr,
		})

		digest, err := name.NewDigest(repoToCopyName + "@" + nonDistributableLayerDigest)
		if err != nil {
			t.Fatalf("Unable to determine the digest of the non-distributable layer. Got: %v", err)
		}

		layer, err := remote.Layer(digest, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if err != nil {
			t.Fatalf("Unable to fetch the layer of the copied image: %v", err)
		}

		_, err = layer.Compressed()
		if err != nil {
			t.Fatalf("Expected to find a non-distributable layer however it wasn't found. Got response code: %v", err)
		}

		imageDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, stderr.String()))

		imgpkg.Run([]string{"pull", "-i", repoToCopyName + imageDigest, "--output", env.Assets.CreateTempFolder("pulled-image")})
	})

	t.Run("Without --include-non-distributable flag and image contains a non-distributable layer should only copy distributable layers and print a warning message", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		airgappedRepo := startRegistryForAirgapTesting(t, env)

		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
		defer env.Cleanup()

		// general setup
		testDir := env.Assets.CreateTempFolder("image-to-tar")
		tarFilePath := filepath.Join(testDir, "image.tar")

		nonDistributableLayerDigest := env.ImageFactory.PushImageWithANonDistributableLayer(airgappedRepo)
		repoToCopyName := env.RelocationRepo + "include-non-distributable"

		// copy to tar
		imgpkg.Run([]string{"copy", "-i", airgappedRepo, "--to-tar", tarFilePath, "--include-non-distributable"})

		stopRegistryForAirgapTesting(t, env)

		var stdOutWriter bytes.Buffer
		imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName}, helpers.RunOpts{
			StdoutWriter: &stdOutWriter,
			StderrWriter: &stdOutWriter,
		})

		if !strings.Contains(stdOutWriter.String(), "Skipped layer due to it being non-distributable.") {
			t.Fatalf("Expected warning message to user, specifying which layer was skipped. But found: %s", stdOutWriter.String())
		}
		digestOfNonDistributableLayer, err := name.NewDigest(repoToCopyName + "@" + nonDistributableLayerDigest)
		if err != nil {
			t.Fatalf("Unable to determine the digest of the non-distributable layer. Got: %v", err)
		}

		layer, err := remote.Layer(digestOfNonDistributableLayer, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if err != nil {
			t.Fatalf("Unable to fetch the layer of the copied image: %v", err)
		}

		_, err = layer.Compressed()
		if err == nil {
			t.Fatalf("Expected non-distributable layer to NOT be copied into registry, however it was")
		}
	})

	t.Run("With --lock-output flag should generate a valid ImageLock file", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
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
		if err := env.Assert.ValidateImagesPresenceInRegistry(refs); err != nil {
			t.Fatalf("could not validate image presence: %v", err)
		}
	})
}

func TestCopyErrorsWhenCopyImageUsingBundleFlag(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	// create generic image
	out := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
	imageDigest := fmt.Sprintf("@%s", helpers.ExtractDigest(t, out))
	imageDigestRef := env.Image + imageDigest

	var stderrBs bytes.Buffer
	_, err := imgpkg.RunWithOpts([]string{"copy", "-b", imageDigestRef, "--to-tar", "fake_path"},
		helpers.RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	if err == nil {
		t.Fatalf("Expected incorrect flag error")
	}

	if !strings.Contains(errOut, "Expected bundle image but found plain image (hint: Did you use -i instead of -b?)") {
		t.Fatalf("Expected error to contain message about using the wrong copy flag, got: %s", errOut)
	}
}

func TestCopyErrorsWhenCopyToTarAndGenerateOutputLockFile(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{t, helpers.Logger{}, env.ImgpkgPath}
	_, err := imgpkg.RunWithOpts(
		[]string{"copy", "--tty", "-i", env.Image, "--to-tar", "file", "--lock-output", "bogus"},
		helpers.RunOpts{AllowError: true},
	)

	if err == nil || !strings.Contains(err.Error(), "output lock file with tar destination") {
		t.Fatalf("expected copy to fail when --lock-output is provided with a tar destination, got %v", err)
	}
}

func stopRegistryForAirgapTesting(t *testing.T, env *helpers.Env) {
	err := exec.Command("docker", "stop", "registry-for-airgapped-testing").Run()
	if err != nil {
		t.Fatalf(err.Error())
	}

	env.AddCleanup(func() {
		exec.Command("docker", "start", "registry-for-airgapped-testing").Run()
	})
}

func startRegistryForAirgapTesting(t *testing.T, env *helpers.Env) string {
	dockerRunCmd := exec.Command("docker", "run", "-d", "-p", "5000", "--env", "REGISTRY_VALIDATION_MANIFESTS_URLS_ALLOW=- ^https?://", "--restart", "always", "--name", "registry-for-airgapped-testing", "registry:2")
	output, err := dockerRunCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("output: %s, %s", output, err)
	}

	env.AddCleanup(func() {
		exec.Command("docker", "stop", "registry-for-airgapped-testing").Run()
		exec.Command("docker", "rm", "registry-for-airgapped-testing").Run()
	})

	inspectCmd := exec.Command("docker", "inspect", `--format='{{(index (index .NetworkSettings.Ports "5000/tcp") 0).HostPort}}'`, "registry-for-airgapped-testing")
	output, err = inspectCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("output: %s, %s", output, err)
	}

	hostPort := strings.ReplaceAll(string(output), "'", "")
	return fmt.Sprintf("localhost:%s/repo/airgapped-image", strings.ReplaceAll(hostPort, "\n", ""))
}
