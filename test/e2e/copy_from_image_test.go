// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

func TestCopyImageToRepoDestinationAndOutputImageLockFileAndPreserverImageTag(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
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

func TestCopyImageInputToTarFileAndToADifferentRepoCheckImageLockIsGenerated(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	// general setup
	testDir := env.Assets.CreateTempFolder("image-to-tar")
	tarFilePath := filepath.Join(testDir, "image.tar")

	// create generic image
	tag := fmt.Sprintf("%d", time.Now().UnixNano())
	tagRef := fmt.Sprintf("%s:%s", env.Image, tag)
	out := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, tagRef)
	imageDigest := fmt.Sprintf("@%s", extractDigest(t, out))

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
}

func TestCopyImageInputToTarWithNonDistributableLayersFlagButContainsANonDistributableLayer(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	// general setup
	testDir := env.Assets.CreateTempFolder("image-to-tar")
	tarFilePath := filepath.Join(testDir, "image.tar")

	nonDistributableLayerDigest := env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo)
	repoToCopyName := env.RelocationRepo + "include-non-distributable"

	// copy to tar
	imgpkg.Run([]string{"copy", "-i", env.RelocationRepo, "--to-tar", tarFilePath, "--include-non-distributable"})

	imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName, "--include-non-distributable"})

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
}

func TestCopyImageInputToTarWithoutNonDistributableLayersFlagButContainsANonDistributableLayer(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	// general setup
	testDir := env.Assets.CreateTempFolder("image-to-tar")
	tarFilePath := filepath.Join(testDir, "image.tar")

	nonDistributableLayerDigest := env.ImageFactory.PushImageWithANonDistributableLayer(env.RelocationRepo)
	repoToCopyName := env.RelocationRepo + "include-non-distributable"

	// copy to tar
	imgpkg.Run([]string{"copy", "-i", env.RelocationRepo, "--to-tar", tarFilePath, "--include-non-distributable"})

	var stdOutWriter bytes.Buffer
	imgpkg.RunWithOpts([]string{"copy", "--tar", tarFilePath, "--to-repo", repoToCopyName}, RunOpts{
		StdoutWriter: &stdOutWriter,
		StderrWriter: &stdOutWriter,
	})

	if !strings.Contains(stdOutWriter.String(), "Skipped layer") {
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
}

func TestCopyErrorsWhenCopyImageUsingBundleFlag(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	defer env.Cleanup()

	// create generic image
	out := env.ImageFactory.PushSimpleAppImageWithRandomFile(imgpkg, env.Image)
	imageDigest := fmt.Sprintf("@%s", extractDigest(t, out))
	imageDigestRef := env.Image + imageDigest

	var stderrBs bytes.Buffer
	_, err := imgpkg.RunWithOpts([]string{"copy", "-b", imageDigestRef, "--to-tar", "fake_path"},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	if err == nil {
		t.Fatalf("Expected incorrect flag error")
	}

	if !strings.Contains(errOut, "Expected bundle image but found plain image (hint: Did you use -i instead of -b?)") {
		t.Fatalf("Expected error to contain message about using the wrong copy flag, got: %s", errOut)
	}
}

func TestCopyErrorsWhenCopyToTarAndGenerateOutputLockFile(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}
	_, err := imgpkg.RunWithOpts(
		[]string{"copy", "--tty", "-i", env.Image, "--to-tar", "file", "--lock-output", "bogus"},
		RunOpts{AllowError: true},
	)

	if err == nil || !strings.Contains(err.Error(), "output lock file with tar destination") {
		t.Fatalf("expected copy to fail when --lock-output is provided with a tar destination, got %v", err)
	}
}
