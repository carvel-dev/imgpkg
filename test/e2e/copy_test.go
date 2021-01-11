// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
)

func TestCopyWithBundleLockInputToRepoDestinationUsingGCloudWithAnExpiredToken(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-bundleLock-repo")
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image
	imageLockYAML := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
 - annotations:
     kbld.carvel.dev/id: gcr.io/cf-k8s-lifecycle-tooling-klt/kpack-build-init@sha256:8136ff3a64517457b91f86bf66b8ffe13b986aaf3511887eda107e59dcb8c632
   image: gcr.io/cf-k8s-lifecycle-tooling-klt/kpack-build-init@sha256:8136ff3a64517457b91f86bf66b8ffe13b986aaf3511887eda107e59dcb8c632
 - annotations:
     kbld.carvel.dev/id: gcr.io/cf-k8s-lifecycle-tooling-klt/nginx@sha256:f35b49b1d18e083235015fd4bbeeabf6a49d9dc1d3a1f84b7df3794798b70c13
   image: gcr.io/cf-k8s-lifecycle-tooling-klt/nginx@sha256:f35b49b1d18e083235015fd4bbeeabf6a49d9dc1d3a1f84b7df3794798b70c13
 - annotations:
     kbld.carvel.dev/id: gcr.io/cf-k8s-lifecycle-tooling-klt/kpack-completion@sha256:1e83c4ccb56ad3e0fccbac74f91dfc404db280f8d3380cfa20c7d68fd0359235
   image: gcr.io/cf-k8s-lifecycle-tooling-klt/kpack-completion@sha256:1e83c4ccb56ad3e0fccbac74f91dfc404db280f8d3380cfa20c7d68fd0359235
`

	// create a bundle with ref to generic
	_, err = createBundleDir(testDir, bundleYAML, imageLockYAML)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}

	// create bundle that refs image with --lock-ouput and a random tag based on time
	imgpkg.Run([]string{"push", "-b", fmt.Sprintf("%s:%v", env.Image, time.Now().UnixNano()), "-f", testDir, "--lock-output", lockFile})

	lockOutputPath := filepath.Join(os.TempDir(), "bundle-lock-relocate-lock.yml")
	defer os.Remove(lockOutputPath)

	homeDir, _ := os.UserHomeDir()
	dockerConfigPath := filepath.Join(homeDir, ".docker/config.json")
	originalDockerConfigJSONContents, err := ioutil.ReadFile(dockerConfigPath)
	if err != nil {
		t.Fatalf("failed to read docker config: %v", err)
	}

	exec.Command("docker", "pull", "ubuntu:21.04").Run()
	defer exec.Command("docker", "volume", "rm", "volume-to-use-when-locking").Run()

	var dockerConfigJSONMap map[string]interface{}
	err = json.Unmarshal(originalDockerConfigJSONContents, &dockerConfigJSONMap)
	if err != nil {
		t.Fatalf("failed to unmarshal docker config.json: %v", err)
	}
	dockerConfigJSONMap["credHelpers"] = map[string]string{"gcr.io": "gcloud-race-condition-db-error"}

	dockerConfigJSONContents, err := json.Marshal(dockerConfigJSONMap)
	if err != nil {
		t.Fatalf("failed to marshal new docker config.json: %v", err)
	}

	err = ioutil.WriteFile(dockerConfigPath, dockerConfigJSONContents, os.ModePerm)
	if err != nil {
		t.Fatalf("failed to write docker config: %v", err)
	}

	defer ioutil.WriteFile(dockerConfigPath, originalDockerConfigJSONContents, os.ModePerm)

	dir, err := filepath.Abs("./")
	if err != nil {
		t.Fatalf("failed to get directory of current file: %v", err)
	}

	// copy via output file
	imgpkg.RunWithOpts([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath}, RunOpts{
		EnvVars: []string{fmt.Sprintf("PATH=%s:%s", os.Getenv("PATH"), filepath.Join(dir, "assets"))},
	})
}

func TestCopyWithBundleLockInputWithIndexesToRepoDestinationAndOutputNewBundleLockFile(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-with-index")
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image
	imageLockYAML := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
 - annotations:
     kbld.carvel.dev/id: index.docker.io/library/nginx@sha256:4cf620a5c81390ee209398ecc18e5fb9dd0f5155cd82adcbae532fec94006fb9
   image: index.docker.io/library/nginx@sha256:4cf620a5c81390ee209398ecc18e5fb9dd0f5155cd82adcbae532fec94006fb9
`

	// create a bundle with ref to generic
	_, err = createBundleDir(testDir, bundleYAML, imageLockYAML)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}

	// create bundle that refs image with --lock-output and a random tag based on time
	imgpkg.Run([]string{"push", "-b", fmt.Sprintf("%s:%v", env.Image, time.Now().UnixNano()), "-f", testDir, "--lock-output", lockFile})

	lockOutputPath := filepath.Join(os.TempDir(), "bundle-lock-relocate-lock.yml")
	defer os.Remove(lockOutputPath)

	dir, err := filepath.Abs("./")
	if err != nil {
		t.Fatalf("failed to get directory of current file: %v", err)
	}

	// copy via output file
	imgpkg.RunWithOpts([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath}, RunOpts{
		EnvVars: []string{fmt.Sprintf("PATH=%s:%s", os.Getenv("PATH"), filepath.Join(dir, "assets"))},
	})

	// check if nginx Index image is present in the repository
	refs := []string{
		env.RelocationRepo + "@sha256:8136ff3a64517457b91f86bf66b8ffe13b986aaf3511887eda107e59dcb8c632",
	}
	if err := validateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyWithBundleLockInputToRepoDestinationAndOutputNewBundleLockFile(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-bundleLock-repo")
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image
	assetsPath := filepath.Join("assets", "simple-app")

	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigest

	imgLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgLockYAML)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	// create bundle that refs image with --lock-ouput and a random tag based on time
	imgpkg.Run([]string{"push", "-b", fmt.Sprintf("%s:%v", env.Image, time.Now().UnixNano()), "-f", assetsPath, "--lock-output", lockFile})
	bundleLock, err := lockconfig.NewBundleLockFromPath(lockFile)
	if err != nil {
		t.Fatalf("failed to read bundlelock file: %v", err)
	}
	bundleDigest := fmt.Sprintf("@%s", extractDigest(bundleLock.Bundle.Image, t))
	bundleTag := bundleLock.Bundle.Tag

	lockOutputPath := filepath.Join(os.TempDir(), "bundle-lock-relocate-lock.yml")
	defer os.Remove(lockOutputPath)

	// copy via output file
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	relocatedBundleLock, err := lockconfig.NewBundleLockFromPath(lockOutputPath)
	if err != nil {
		t.Fatalf("unable to read relocated bundle lock file: %s", err)
	}

	relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
	if relocatedBundleLock.Bundle.Image != relocatedRef {
		t.Fatalf("expected bundle digest to be '%s', but was '%s'", relocatedRef, relocatedBundleLock.Bundle.Image)
	}

	if relocatedBundleLock.Bundle.Tag != bundleTag {
		t.Fatalf("expected bundle tag to have tag '%v', was '%s'", bundleTag, relocatedBundleLock.Bundle.Tag)
	}

	if err := validateBundleLockApiVersionAndKind(relocatedBundleLock); err != nil {
		t.Fatal(err.Error())
	}

	// check if bundle and referenced images are present in dst repo
	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest, env.RelocationRepo + ":" + bundleTag}
	if err := validateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyWithImageLockInputToRepoDestinationAndOutputNewImageLockFile(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-imageLock-repo")
	lockFile := filepath.Join(testDir, "images.lock.yml")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image
	assetsPath := filepath.Join("assets", "simple-app")

	// force digest to change so test is meaningful
	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":v1", "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	err = ioutil.WriteFile(lockFile, []byte(imageLockYAML), 0700)
	if err != nil {
		t.Fatalf("failed to create images.lock file: %v", err)
	}

	lockOutputPath := filepath.Join(os.TempDir(), "image-relocate-lock.yml")
	defer os.Remove(lockOutputPath)

	// copy via output file
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	relocateImgLock, err := lockconfig.NewImagesLockFromPath(lockOutputPath)

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	if len(relocateImgLock.Images) != 1 {
		t.Fatalf("expected 1 image in the lock file but found %d", len(relocateImgLock.Images))
	}
	if relocateImgLock.Images[0].Image != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'", expectedRef, relocateImgLock.Images[0].Image)
	}

	if err := validateImageLockApiVersionAndKind(relocateImgLock); err != nil {
		t.Fatal(err.Error())
	}

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigest}
	if err := validateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyBundleWithCollocatedReferencedImagesToRepoDestinationAndOutputBundleLockFile(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	assetsPath := filepath.Join("assets", "simple-app")

	// force digest to change so test is meaningful
	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s%s
`, env.Image, imageDigest)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imageLockYAML)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	// create bundle that refs image and a random tag based on time
	bundleTag := fmt.Sprintf(":%d", time.Now().UnixNano())
	out = imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s%s", env.Image, bundleTag), "-f", assetsPath})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(out, t))

	lockOutputPath := filepath.Join(os.TempDir(), "bundle-relocate-lock.yml")
	defer os.Remove(lockOutputPath)
	// copy via created ref
	imgpkg.Run([]string{"copy",
		"--bundle", fmt.Sprintf("%s%s", env.Image, bundleTag),
		"--to-repo", env.RelocationRepo,
		"--lock-output", lockOutputPath},
	)

	relocatedBundleLock, err := lockconfig.NewBundleLockFromPath(lockOutputPath)
	if err != nil {
		t.Fatalf("unable to read bundle lock file: %s", err)
	}

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
	if relocatedBundleLock.Bundle.Image != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'", expectedRef, relocatedBundleLock.Bundle.Image)
	}

	if trimmedTag := strings.TrimPrefix(bundleTag, ":"); relocatedBundleLock.Bundle.Tag != trimmedTag {
		t.Fatalf("expected lock output to contain tag '%s', got '%s'", trimmedTag, relocatedBundleLock.Bundle.Tag)
	}

	if err := validateBundleLockApiVersionAndKind(relocatedBundleLock); err != nil {
		t.Fatal(err.Error())
	}

	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleTag, env.RelocationRepo + bundleDigest}
	if err := validateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyBundleWithNonCollocatedReferencedImagesToRepoDestination(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	assetsPath := filepath.Join("assets", "simple-app")

	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	image := env.Image + "-image-outside-repo"
	out := imgpkg.Run([]string{"push", "--tty", "-i", image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	// image intentionally does not exist in bundle repo
	imageDigestRef := image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imageLockYAML)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	out = imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", assetsPath})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	bundleDigestRef := env.Image + bundleDigest

	imgpkg.Run([]string{"copy", "--bundle", bundleDigestRef, "--to-repo", env.RelocationRepo})

	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest}
	if err := validateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyImageToRepoDestinationAndOutputImageLockFile(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// create generic image
	assetsPath := filepath.Join("assets", "simple-app")

	// force digest to change so test is meaningful
	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	tag := time.Now().UnixNano()
	out := imgpkg.Run([]string{"push", "--tty", "-i", fmt.Sprintf("%s:%v", env.Image, tag), "-f", assetsPath})
	imageDigestTag := fmt.Sprintf("@%s", extractDigest(out, t))

	lockOutputPath := filepath.Join(os.TempDir(), "image-relocate-lock.yml")
	defer os.Remove(lockOutputPath)

	// copy via create ref
	imgpkg.Run([]string{"copy", "--image", fmt.Sprintf("%s:%v", env.Image, tag),
		"--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	imgLock, err := lockconfig.NewImagesLockFromPath(lockOutputPath)
	if err != nil {
		t.Fatalf("unable to read the image lock file: %s", err)
	}

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigestTag)
	if imgLock.Images[0].Image != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'",
			expectedRef, imgLock.Images[0].Image)
	}

	if err := validateImageLockApiVersionAndKind(imgLock); err != nil {
		t.Fatal(err.Error())
	}

	if err := validateImagesPresenceInRegistry([]string{env.RelocationRepo + imageDigestTag}); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}

	if err := validateImagesPresenceInRegistry([]string{fmt.Sprintf("%s:%v", env.RelocationRepo, tag)}); err == nil {
		t.Fatalf("expected not to find image with tag '%v', but did", tag)
	}
}

func TestCopyWithBundleLockInputToTarFileAndToADifferentRepoCheckTagIsKeptAndBundleLockFileIsGenerated(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-bundleLock-tar")
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	tarFilePath := filepath.Join(testDir, "bundle.tar")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image

	assetsPath := filepath.Join("assets", "simple-app")

	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imageLockYAML)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	// create bundle that refs image with --lock-ouput
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsPath, "--lock-output", lockFile})

	origBundleLock, err := lockconfig.NewBundleLockFromPath(lockFile)
	if err != nil {
		t.Fatalf("unable to read original bundle lock: %s", err)
	}

	bundleDigestRef := fmt.Sprintf("%s@%s", env.Image, extractDigest(origBundleLock.Bundle.Image, t))

	// copy via output file
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-tar", tarFilePath})

	imagesOrIndexes, err := imagetar.NewTarReader(tarFilePath).Read()
	if err != nil {
		t.Fatalf("failed to read tar: %v", err)
	}

	for _, imageOrIndex := range imagesOrIndexes {
		imageRefFromTar := imageOrIndex.Ref()
		if !(imageRefFromTar == imageDigestRef || imageRefFromTar == bundleDigestRef) {
			t.Fatalf("unexpected image ref (%s) referenced in manifest.json", imageRefFromTar)
		}
	}

	lockFilePath := filepath.Join(os.TempDir(), "relocate-from-tar-lock.yml")
	defer os.Remove(lockFilePath)

	// copy from tar to repo
	imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})

	relocatedBundleLock, err := lockconfig.NewBundleLockFromPath(lockFilePath)
	if err != nil {
		t.Fatalf("could not read relocated bundle lock file: %v", err)
	}

	expectedRelocatedRef := fmt.Sprintf("%s@%s", env.RelocationRepo, extractDigest(bundleDigestRef, t))
	if relocatedBundleLock.Bundle.Image != expectedRelocatedRef {
		t.Fatalf("expected bundle digest to be '%s', but was '%s'", expectedRelocatedRef, relocatedBundleLock.Bundle.Image)
	}

	if origBundleLock.Bundle.Tag != relocatedBundleLock.Bundle.Tag {
		t.Fatalf("expected bundle tag to have tag '%v', was '%s'",
			origBundleLock.Bundle.Tag, relocatedBundleLock.Bundle.Tag)
	}

	if err := validateBundleLockApiVersionAndKind(relocatedBundleLock); err != nil {
		t.Fatal(err.Error())
	}

	// validate bundle and image were relocated
	relocatedBundleRef := expectedRelocatedRef
	relocatedImageRef := env.RelocationRepo + imageDigest
	relocatedBundleTagRef := fmt.Sprintf("%s:%v", env.RelocationRepo, origBundleLock.Bundle.Tag)

	if err := validateImagesPresenceInRegistry([]string{relocatedBundleRef, relocatedImageRef, relocatedBundleTagRef}); err != nil {
		t.Fatalf("Failed to locate digest in relocationRepo: %v", err)
	}
}

func TestCopyWithImageLockInputToTarFileAndToADifferentRepoCheckTagIsKeptAndImageLockFileIsGenerated(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-imageLock-tar")
	lockFile := filepath.Join(testDir, "images.lock.yml")
	tarFilePath := filepath.Join(testDir, "image.tar")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image
	assetsPath := filepath.Join("assets", "simple-app")

	// force digest to change so test is meaningful
	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	err = ioutil.WriteFile(lockFile, []byte(imageLockYAML), 0700)
	if err != nil {
		t.Fatalf("failed to create images.lock file: %v", err)
	}

	// copy via output file
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-tar", tarFilePath})

	imagesOrIndexes, err := imagetar.NewTarReader(tarFilePath).Read()
	if err != nil {
		t.Fatalf("failed to read tar: %v", err)
	}

	for _, imageOrIndex := range imagesOrIndexes {
		imageRefFromTar := imageOrIndex.Ref()
		if !(imageRefFromTar == imageDigestRef) {
			t.Fatalf("unexpected image ref (%s) referenced in manifest.json", imageRefFromTar)
		}
	}

	lockOutputPath := filepath.Join(os.TempDir(), "relocate-from-tar-lock.yml")
	defer os.Remove(lockOutputPath)

	// copy from tar to repo
	imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	imgLock, err := lockconfig.NewImagesLockFromPath(lockOutputPath)
	if err != nil {
		t.Fatalf("could not read lock-output: %v", err)
	}

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	if imgLock.Images[0].Image != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'", expectedRef, imgLock.Images[0].Image)
	}

	if err := validateImageLockApiVersionAndKind(imgLock); err != nil {
		t.Fatal(err.Error())
	}

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigest}
	if err := validateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}

}

func TestCopyBundleToTarFileAndToADifferentRepoCheckTagIsKeptAndBundleLockFileIsGenerated(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-bundle-tar")
	tarFilePath := filepath.Join(testDir, "bundle.tar")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image

	assetsPath := filepath.Join("assets", "simple-app")

	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imageLockYAML)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	tag := time.Now().UnixNano()
	// create bundle that refs image
	out = imgpkg.Run([]string{"push", "--tty", "-b", fmt.Sprintf("%s:%v", env.Image, tag), "-f", assetsPath})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(out, t))

	// copy to a tar
	imgpkg.Run([]string{"copy", "-b", fmt.Sprintf("%s:%v", env.Image, tag), "--to-tar", tarFilePath})

	lockFilePath := filepath.Join(os.TempDir(), "relocate-from-tar-lock.yml")
	defer os.Remove(lockFilePath)

	// copy from tar to repo
	imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})

	bundleLock, err := lockconfig.NewBundleLockFromPath(lockFilePath)
	if err != nil {
		t.Fatalf("could not read lock file: %v", err)
	}

	relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
	if bundleLock.Bundle.Image != relocatedRef {
		t.Fatalf("expected bundle digest to be '%s', but was '%s'", relocatedRef, bundleLock.Bundle.Image)
	}

	if bundleLock.Bundle.Tag != fmt.Sprintf("%v", tag) {
		t.Fatalf("expected bundle tag to have tag '%v', was '%s'", tag, bundleLock.Bundle.Tag)
	}

	if err := validateBundleLockApiVersionAndKind(bundleLock); err != nil {
		t.Fatal(err.Error())
	}

	// validate bundle and image were relocated
	relocatedBundleRef := env.RelocationRepo + bundleDigest
	relocatedImageRef := env.RelocationRepo + imageDigest
	relocatedBundleTagRef := fmt.Sprintf("%s:%v", env.RelocationRepo, tag)

	if err := validateImagesPresenceInRegistry([]string{relocatedBundleRef, relocatedImageRef, relocatedBundleTagRef}); err != nil {
		t.Fatalf("Failed to locate digest in relocationRepo: %v", err)
	}
}

func TestCopyImageInputToTarFileAndToADifferentRepoCheckImageLockIsGenerated(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-image-tar")
	tarFilePath := filepath.Join(testDir, "image.tar")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image
	assetsPath := filepath.Join("assets", "simple-app")

	// force digest to change so test is meaningful
	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	tag := fmt.Sprintf("%d", time.Now().UnixNano())
	tagRef := fmt.Sprintf("%s:%s", env.Image, tag)
	out := imgpkg.Run([]string{"push", "--tty", "-i", tagRef, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))

	// copy to tar
	imgpkg.Run([]string{"copy", "-i", tagRef, "--to-tar", tarFilePath})

	lockOutputPath := filepath.Join(os.TempDir(), "relocate-from-tar-lock.yml")
	defer os.Remove(lockOutputPath)

	// copy from tar to repo
	imgpkg.Run([]string{"copy", "--tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	imgLock, err := lockconfig.NewImagesLockFromPath(lockOutputPath)
	if err != nil {
		t.Fatalf("could not read lock-output: %v", err)
	}

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	if imgLock.Images[0].Image != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'", imgLock.Images[0].Image, expectedRef)
	}

	if err := validateImageLockApiVersionAndKind(imgLock); err != nil {
		t.Fatal(err.Error())
	}

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigest}
	if err := validateImagesPresenceInRegistry(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyErrorsWhenCopyImageUsingBundleFlag(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// create generic image
	assetsPath := filepath.Join("assets", "simple-app")

	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigest

	var stderrBs bytes.Buffer
	_, err = imgpkg.RunWithOpts([]string{"copy", "-b", imageDigestRef, "--to-tar", "fake_path"},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	if err == nil {
		t.Fatalf("Expected incorrect flag error")
	}

	if !strings.Contains(errOut, "Expected bundle image but found plain image (hint: Did you use -i instead of -b?)") {
		t.Fatalf("Expected error to contain message about using the wrong copy flag, got: %s", errOut)
	}
}

func TestCopyErrorsWhenCopyBundleUsingImageFlag(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// create generic image
	assetsPath := filepath.Join("assets", "simple-app")

	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imageLockYAML)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	out = imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", assetsPath})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	bundleDigestRef := env.Image + bundleDigest

	var stderrBs bytes.Buffer
	_, err = imgpkg.RunWithOpts([]string{"copy", "-i", bundleDigestRef, "--to-tar", "fake_path"},
		RunOpts{AllowError: true, StderrWriter: &stderrBs})
	errOut := stderrBs.String()

	if err == nil {
		t.Fatalf("Expected incorrect flag error")
	}
	if !strings.Contains(errOut, "Expected bundle flag when copying a bundle (hint: Use -b instead of -i for bundles)") {
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

func TestCopyBundlePreserveImgLockAnnotations(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}, env.ImgpkgPath}

	// general setup
	testDir := filepath.Join(os.TempDir(), "imgpkg-test-copy-bundleLock-repo")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// create generic image
	assetsPath := filepath.Join("assets", "simple-app")

	randFile, err := addRandomFile(assetsPath)
	if err != nil {
		t.Fatalf("failed to create unuique file: %v", err)
	}
	defer os.Remove(randFile)

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigest

	imageLockYAML := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: %s
  annotations:
    greeting: hello world
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imageLockYAML)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	// create bundle that refs image with --lock-ouput and a random tag based on time
	out = imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", assetsPath})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	bundleDigestRef := env.Image + bundleDigest

	// copy
	imgpkg.Run([]string{"copy", "-b", bundleDigestRef, "--to-repo", env.RelocationRepo})

	// pull
	bundleDigestRef = env.RelocationRepo + bundleDigest
	imgpkg.Run([]string{"pull", "-b", bundleDigestRef, "-o", testDir})

	imgLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(testDir, ".imgpkg", "images.yml"))
	if err != nil {
		t.Fatalf("could not read images lock: %v", err)
	}

	greeting, ok := imgLock.Images[0].Annotations["greeting"]
	if !ok {
		t.Fatalf("could not find annoation greeting in images lock")
	}
	if greeting != "hello world" {
		t.Fatalf("Expected images lock to have annotation saying 'hello world', got: %s", greeting)
	}

}

func addRandomFile(dir string) (string, error) {
	randFile := filepath.Join(dir, "rand.yml")
	randContents := fmt.Sprintf("%d", time.Now().UnixNano())
	err := ioutil.WriteFile(randFile, []byte(randContents), 0700)
	if err != nil {
		return "", err
	}

	return randFile, nil
}

func validateImagesPresenceInRegistry(refs []string) error {
	for _, refString := range refs {
		ref, _ := name.ParseReference(refString)
		if _, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
			return fmt.Errorf("validating image %s: %v", refString, err)
		}
	}
	return nil
}

func validateBundleLockApiVersionAndKind(bLock lockconfig.BundleLock) error {
	// Do not replace bundleLockKind or bundleLockAPIVersion with consts
	// BundleLockKind or BundleLockAPIVersion.
	// This is done to prevent updating the const.
	bundleLockKind := "BundleLock"
	bundleLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if bLock.APIVersion != bundleLockAPIVersion {
		return fmt.Errorf("expected apiVersion to equal: %s, but got: %s", bundleLockAPIVersion, bLock.APIVersion)
	}

	if bLock.Kind != bundleLockKind {
		return fmt.Errorf("expected Kind to equal: %s, but got: %s", bundleLockKind, bLock.Kind)
	}
	return nil
}

func validateImageLockApiVersionAndKind(iLock lockconfig.ImagesLock) error {
	// Do not replace imageLockKind or imagesLockAPIVersion with consts
	// ImagesLockKind or ImagesLockAPIVersion.
	// This is done to prevent updating the const.
	imagesLockKind := "ImagesLock"
	imagesLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if iLock.APIVersion != imagesLockAPIVersion {
		return fmt.Errorf("expected apiVersion to equal: %s, but got: %s", imagesLockAPIVersion, iLock.APIVersion)
	}

	if iLock.Kind != imagesLockKind {
		return fmt.Errorf("expected Kind to equal: %s, but got: %s", imagesLockKind, iLock.Kind)
	}
	return nil
}
