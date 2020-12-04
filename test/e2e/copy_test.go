// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"gopkg.in/yaml.v2"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	lf "github.com/k14s/imgpkg/pkg/imgpkg/lockfiles"
)

func TestCopyBundleLockInputToRepoWithLockOutput(t *testing.T) {
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

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	// create bundle that refs image with --lock-ouput and a random tag based on time
	imgpkg.Run([]string{"push", "-b", fmt.Sprintf("%s:%v", env.Image, time.Now().UnixNano()), "-f", assetsPath, "--lock-output", lockFile})
	bundleLockYml, err := lf.ReadBundleLockFile(lockFile)
	if err != nil {
		t.Fatalf("failed to read bundlelock file: %v", err)
	}
	bundleDigest := fmt.Sprintf("@%s", extractDigest(bundleLockYml.Spec.Image.DigestRef, t))
	bundleTag := bundleLockYml.Spec.Image.OriginalTag

	lockOutputPath := filepath.Join(os.TempDir(), "bundle-lock-relocate-lock.yml")
	defer os.Remove(lockOutputPath)

	// copy via output file
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	bLockBytes, err := ioutil.ReadFile(lockOutputPath)
	if err != nil {
		t.Fatalf("could not read lock-output: %v", err)
	}

	var bLock lf.BundleLock
	err = yaml.Unmarshal(bLockBytes, &bLock)
	if err != nil {
		t.Fatalf("could not unmarshal lock output: %v", err)
	}

	relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
	if bLock.Spec.Image.DigestRef != relocatedRef {
		t.Fatalf("expected bundle digest to be '%s', but was '%s'", relocatedRef, bLock.Spec.Image.DigestRef)
	}

	if bLock.Spec.Image.OriginalTag != bundleTag {
		t.Fatalf("expected bundle tag to have tag '%v', was '%s'", bundleTag, bLock.Spec.Image.OriginalTag)
	}

	if err := validateBundleLockApiVersionAndKind(bLock); err != nil {
		t.Fatal(err.Error())
	}

	// check if bundle and referenced images are present in dst repo
	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest, env.RelocationRepo + ":" + bundleTag}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyImageLockInputToRepoWithLockOutput(t *testing.T) {
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

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: %s
`, imageDigestRef)

	err = ioutil.WriteFile(lockFile, []byte(imgsYml), 0700)
	if err != nil {
		t.Fatalf("failed to create images.lock file: %v", err)
	}

	lockOutputPath := filepath.Join(os.TempDir(), "image-relocate-lock.yml")
	defer os.Remove(lockOutputPath)

	// copy via output file
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	iLockBytes, err := ioutil.ReadFile(lockOutputPath)
	if err != nil {
		t.Fatalf("could not read lock-output: %v", err)
	}

	var iLock lf.ImageLock
	err = yaml.Unmarshal(iLockBytes, &iLock)
	if err != nil {
		t.Fatalf("could not unmarshal lock output: %v", err)
	}

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	if iLock.Spec.Images[0].Image != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'", expectedRef, iLock.Spec.Images[0].Image)
	}

	if err := validateImageLockApiVersionAndKind(iLock); err != nil {
		t.Fatal(err.Error())
	}

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigest}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyBundleWithCollocatedReferencedImagesToRepoWithLockOutput(t *testing.T) {
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

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: %s%s
`, env.Image, imageDigest)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
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

	bLockBytes, err := ioutil.ReadFile(lockOutputPath)
	if err != nil {
		t.Fatalf("could not read lock-output: %v", err)
	}

	var bLock lf.BundleLock
	err = yaml.Unmarshal(bLockBytes, &bLock)
	if err != nil {
		t.Fatalf("could not unmarshal lock output: %v", err)
	}

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
	if bLock.Spec.Image.DigestRef != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'", expectedRef, bLock.Spec.Image.DigestRef)
	}

	if trimmedTag := strings.TrimPrefix(bundleTag, ":"); bLock.Spec.Image.OriginalTag != trimmedTag {
		t.Fatalf("expected lock output to contain tag '%s', got '%s'", trimmedTag, bLock.Spec.Image.OriginalTag)
	}

	if err := validateBundleLockApiVersionAndKind(bLock); err != nil {
		t.Fatal(err.Error())
	}

	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleTag, env.RelocationRepo + bundleDigest}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyBundleWithNonCollocatedReferencedImagesToRepo(t *testing.T) {
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

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: %s
`, imageDigestRef)

	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	out = imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", assetsPath})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	bundleDigestRef := env.Image + bundleDigest

	imgpkg.Run([]string{"copy", "--bundle", bundleDigestRef, "--to-repo", env.RelocationRepo})

	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyImageInputToRepoWithLockOutput(t *testing.T) {
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
	imgpkg.Run([]string{"copy", "--image", fmt.Sprintf("%s:%v", env.Image, tag), "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	iLockBytes, err := ioutil.ReadFile(lockOutputPath)
	if err != nil {
		t.Fatalf("could not read lock-output: %v", err)
	}

	var iLock lf.ImageLock
	err = yaml.Unmarshal(iLockBytes, &iLock)
	if err != nil {
		t.Fatalf("could not unmarshal lock output: %v", err)
	}

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigestTag)
	if iLock.Spec.Images[0].Image != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'", expectedRef, iLock.Spec.Images[0].Image)
	}

	if err := validateImageLockApiVersionAndKind(iLock); err != nil {
		t.Fatal(err.Error())
	}

	if err := validateImagePresence([]string{env.RelocationRepo + imageDigestTag}); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}

	if err := validateImagePresence([]string{fmt.Sprintf("%s:%v", env.RelocationRepo, tag)}); err == nil {
		t.Fatalf("expected not to find image with tag '%v', but did", tag)
	}
}

func TestCopyBundleLockInputViaTarWithTagAndLockOutput(t *testing.T) {
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

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	// create bundle that refs image with --lock-ouput
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsPath, "--lock-output", lockFile})
	bundlePushLockYml, err := lf.ReadBundleLockFile(lockFile)
	if err != nil {
		t.Fatalf("failed to read bundlelock file: %v", err)
	}
	bundleDigestRef := fmt.Sprintf("%s@%s", env.Image, extractDigest(bundlePushLockYml.Spec.Image.DigestRef, t))

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
	imgpkg.Run([]string{"copy", "--from-tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})

	bLockBytes, err := ioutil.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("could not read lock file: %v", err)
	}

	var bundleLock lf.BundleLock
	err = yaml.Unmarshal(bLockBytes, &bundleLock)
	if err != nil {
		t.Fatalf("could not unmarshal bundleLock")
	}

	relocatedRef := fmt.Sprintf("%s@%s", env.RelocationRepo, extractDigest(bundlePushLockYml.Spec.Image.DigestRef, t))
	if bundleLock.Spec.Image.DigestRef != relocatedRef {
		t.Fatalf("expected bundle digest to be '%s', but was '%s'", relocatedRef, bundleLock.Spec.Image.DigestRef)
	}

	if bundleLock.Spec.Image.OriginalTag != bundlePushLockYml.Spec.Image.OriginalTag {
		t.Fatalf("expected bundle tag to have tag '%v', was '%s'", bundlePushLockYml.Spec.Image.OriginalTag, bundleLock.Spec.Image.OriginalTag)
	}

	if err := validateBundleLockApiVersionAndKind(bundleLock); err != nil {
		t.Fatal(err.Error())
	}

	// validate bundle and image were relocated
	relocatedBundleRef := relocatedRef
	relocatedImageRef := env.RelocationRepo + imageDigest
	relocatedBundleTagRef := fmt.Sprintf("%s:%v", env.RelocationRepo, bundlePushLockYml.Spec.Image.OriginalTag)

	if err := validateImagePresence([]string{relocatedBundleRef, relocatedImageRef, relocatedBundleTagRef}); err != nil {
		t.Fatalf("Failed to locate digest in relocationRepo: %v", err)
	}
}

func TestCopyImageLockInputViaTarWithTagAndLockOutput(t *testing.T) {
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

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: %s
`, imageDigestRef)

	err = ioutil.WriteFile(lockFile, []byte(imgsYml), 0700)
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
	imgpkg.Run([]string{"copy", "--from-tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	iLockBytes, err := ioutil.ReadFile(lockOutputPath)
	if err != nil {
		t.Fatalf("could not read lock-output: %v", err)
	}

	var iLock lf.ImageLock
	err = yaml.Unmarshal(iLockBytes, &iLock)
	if err != nil {
		t.Fatalf("could not unmarshal lock output: %v", err)
	}

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	if iLock.Spec.Images[0].Image != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'", expectedRef, iLock.Spec.Images[0].Image)
	}

	if err := validateImageLockApiVersionAndKind(iLock); err != nil {
		t.Fatal(err.Error())
	}

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigest}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}

}

func TestCopyBundleViaTarWithTagAndLockOutput(t *testing.T) {
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

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
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
	imgpkg.Run([]string{"copy", "--from-tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockFilePath})

	bLockBytes, err := ioutil.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("could not read lock file: %v", err)
	}

	var bundleLock lf.BundleLock
	err = yaml.Unmarshal(bLockBytes, &bundleLock)
	if err != nil {
		t.Fatalf("could not unmarshal bundleLock")
	}

	relocatedRef := fmt.Sprintf("%s%s", env.RelocationRepo, bundleDigest)
	if bundleLock.Spec.Image.DigestRef != relocatedRef {
		t.Fatalf("expected bundle digest to be '%s', but was '%s'", relocatedRef, bundleLock.Spec.Image.DigestRef)
	}

	if bundleLock.Spec.Image.OriginalTag != fmt.Sprintf("%v", tag) {
		t.Fatalf("expected bundle tag to have tag '%v', was '%s'", tag, bundleLock.Spec.Image.OriginalTag)
	}

	if err := validateBundleLockApiVersionAndKind(bundleLock); err != nil {
		t.Fatal(err.Error())
	}

	// validate bundle and image were relocated
	relocatedBundleRef := env.RelocationRepo + bundleDigest
	relocatedImageRef := env.RelocationRepo + imageDigest
	relocatedBundleTagRef := fmt.Sprintf("%s:%v", env.RelocationRepo, tag)

	if err := validateImagePresence([]string{relocatedBundleRef, relocatedImageRef, relocatedBundleTagRef}); err != nil {
		t.Fatalf("Failed to locate digest in relocationRepo: %v", err)
	}
}

func TestCopyImageInputViaTarWithLockOutput(t *testing.T) {
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
	imgpkg.Run([]string{"copy", "--from-tar", tarFilePath, "--to-repo", env.RelocationRepo, "--lock-output", lockOutputPath})

	iLockBytes, err := ioutil.ReadFile(lockOutputPath)
	if err != nil {
		t.Fatalf("could not read lock-output: %v", err)
	}

	var iLock lf.ImageLock
	err = yaml.Unmarshal(iLockBytes, &iLock)
	if err != nil {
		t.Fatalf("could not unmarshal lock output: %v", err)
	}

	expectedRef := fmt.Sprintf("%s%s", env.RelocationRepo, imageDigest)
	if iLock.Spec.Images[0].Image != expectedRef {
		t.Fatalf("expected lock output to contain relocated ref '%s', got '%s'", expectedRef, iLock.Spec.Images[0].Image)
	}

	if err := validateImageLockApiVersionAndKind(iLock); err != nil {
		t.Fatal(err.Error())
	}

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigest}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}

}

func TestCopyBundleWhenImageError(t *testing.T) {
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

	if !strings.Contains(errOut, "Expected image flag when given an image reference. Please run with -i instead of -b, or use -b with a bundle reference") {
		t.Fatalf("Expected error to contain message about using the wrong copy flag, got: %s", errOut)
	}
}

func TestCopyImageWhenBundleError(t *testing.T) {
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

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
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
	if !strings.Contains(errOut, "Expected bundle flag when copying a bundle, please use -b instead of -i") {
		t.Fatalf("Expected error to contain message about using the wrong copy flag, got: %s", errOut)
	}
}

func TestCopyErrorTarDstLockOutput(t *testing.T) {
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

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: %s
    annotations:
      greeting: hello world
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
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

	iLockBytes, err := ioutil.ReadFile(filepath.Join(testDir, lf.BundleDir, imageFile))
	if err != nil {
		t.Fatalf("could not read images lock: %v", err)
	}
	var iLock lf.ImageLock
	err = yaml.Unmarshal(iLockBytes, &iLock)
	if err != nil {
		t.Fatalf("could not unmarshal images lock: %v", err)
	}

	greeting, ok := iLock.Spec.Images[0].Annotations["greeting"]
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

func validateImagePresence(refs []string) error {
	for _, refString := range refs {
		ref, _ := name.ParseReference(refString)
		if _, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain)); err != nil {
			return fmt.Errorf("validating image %s: %v", refString, err)
		}
	}
	return nil
}

func validateBundleLockApiVersionAndKind(bLock lf.BundleLock) error {
	// Do not replace bundleLockKind or bundleLockAPIVersion with consts
	// BundleLockKind or BundleLockAPIVersion.
	// This is done to prevent updating the const.
	bundleLockKind := "BundleLock"
	bundleLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if bLock.ApiVersion != bundleLockAPIVersion {
		return fmt.Errorf("expected apiVersion to equal: %s, but got: %s", bundleLockAPIVersion, bLock.ApiVersion)
	}

	if bLock.Kind != bundleLockKind {
		return fmt.Errorf("expected Kind to equal: %s, but got: %s", bundleLockKind, bLock.Kind)
	}
	return nil
}

func validateImageLockApiVersionAndKind(iLock lf.ImageLock) error {
	// Do not replace imageLockKind or imagesLockAPIVersion with consts
	// ImagesLockKind or ImagesLockAPIVersion.
	// This is done to prevent updating the const.
	imagesLockKind := "ImagesLock"
	imagesLockAPIVersion := "imgpkg.carvel.dev/v1alpha1"
	if iLock.ApiVersion != lf.ImagesLockAPIVersion {
		return fmt.Errorf("expected apiVersion to equal: %s, but got: %s", imagesLockAPIVersion, iLock.ApiVersion)
	}

	if iLock.Kind != imagesLockKind {
		return fmt.Errorf("expected Kind to equal: %s, but got: %s", imagesLockKind, iLock.Kind)
	}
	return nil
}
