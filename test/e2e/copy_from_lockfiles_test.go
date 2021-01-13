// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

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
	imageDigest := fmt.Sprintf("@%s", extractDigest(t, out))
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
	bundleDigest := fmt.Sprintf("@%s", extractDigest(t, bundleLock.Bundle.Image))
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
	imageDigest := fmt.Sprintf("@%s", extractDigest(t, out))
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
	imageDigest := fmt.Sprintf("@%s", extractDigest(t, out))
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

	bundleDigestRef := fmt.Sprintf("%s@%s", env.Image, extractDigest(t, origBundleLock.Bundle.Image))

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

	expectedRelocatedRef := fmt.Sprintf("%s@%s", env.RelocationRepo, extractDigest(t, bundleDigestRef))
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
	imageDigest := fmt.Sprintf("@%s", extractDigest(t, out))
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
