// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package lockfiles_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	lf "github.com/k14s/imgpkg/pkg/imgpkg/lockfiles"
)

func TestIsBundle_ImageIsBundle(t *testing.T) {
	labels := make(map[string]string)
	labels[image.BundleConfigLabel] = "foo"

	img := fakeImage(labels)

	isBundle, err := lf.IsBundle(img)
	if err != nil {
		t.Fatalf("Expected no error but received one: %v", err)
	}

	if !isBundle {
		t.Fatalf("Expected image to be valid Bundle but was not")
	}
}

func TestIsBundle_ImageIsNotBundle(t *testing.T) {
	labels := make(map[string]string)
	img := fakeImage(labels)

	isBundle, err := lf.IsBundle(img)
	if err != nil {
		t.Fatalf("Expected no error but received one: %v", err)
	}

	if isBundle {
		t.Fatalf("Expected image to be invalid Bundle but was not")
	}
}

func TestCollectBundleURLs_ErrorWhenCannotReadBundleLockFile(t *testing.T) {
	_, _, _, err := lf.CollectBundleURLs("/some/invalid/place", &fakeImageRetriever{})
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatal("Expected an error")
	}
	fmt.Println(err)
}

func TestCollectBundleURLs_ErrorWhenOCIImageIsNotABundle(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "bundle-tests")
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	bundleYaml := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: BundleLock
spec:
  image:
    url: somewhere/places
    tag: sometag
`
	err = ioutil.WriteFile(lockFile, []byte(bundleYaml), 0777)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	imgRetriever := &fakeImageRetriever{
		ImageReturns: []imageRetrieverImageReturns{{img: fakeImage(nil), err: nil}},
	}
	_, _, _, err = lf.CollectBundleURLs(lockFile, imgRetriever)
	if err == nil {
		t.Fatalf("unexpected success: %s", err)
	}

	if !strings.Contains(err.Error(), "expected image flag when given an image reference") {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestCollectBundleURLs_SuccessWhenIsABundle_ReturnsBundleImgReferenceTheOriginalTagAndAListOfAllTheImagesInTheBundle(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "bundle-tests")
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	bundleYaml := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: BundleLock
spec:
  image:
    url: somewhere/places
    tag: sometag
`
	err = ioutil.WriteFile(lockFile, []byte(bundleYaml), 0777)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	imgRetriever := &fakeImageRetriever{
		ImageReturns: []imageRetrieverImageReturns{{img: fakeBundleOCIImage(t), err: nil}},
	}

	bundleRef, origTag, imgsInBundle, err := lf.CollectBundleURLs(lockFile, imgRetriever)
	if err != nil {
		t.Fatalf("unexpected success: %s", err)
	}

	if bundleRef.Name() != "index.docker.io/somewhere/places:latest" {
		t.Fatalf("Expecting bundle image to be index.docker.io/somewhere/places:latest but was %s", bundleRef.Name())
	}

	if origTag != "sometag" {
		t.Fatalf("Expecting bundle original tag to remain `sometag` but was %s", bundleRef.Name())
	}

	if len(imgsInBundle) != 1 {
		t.Fatalf("Expecting bundle have 1 image but it has %+v", imgsInBundle)
	}

	if imgsInBundle[0].Name() != "index.docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0" {
		t.Fatalf("Expecting bundle image to be index.docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 but was %s", imgsInBundle[0].Name())
	}
}

func TestCollectImageLockURLs_ErrorWhenCannotReadBundleLockFile(t *testing.T) {
	_, err := lf.CollectImageLockURLs("/some/invalid/place", &fakeImageRetriever{})
	if err == nil || !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatal("Expected an error")
	}
	fmt.Println(err)
}

func TestCollectImageLockURLs_ErrorWhenOneOfTheImagesIsABundle(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "bundle-tests")
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	bundleYaml := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: some/image@sha256:ecde7be1a7f238ea1721f8f8ba5dddae07a53e35eb1bb1f242e97db4aa2bc890
  - image: localhost:5000/some/other_image@sha256:e0c01e71eb67453430e43947b628a40d2cca58ed7154bc33339fc3b0f3189671
`
	err = ioutil.WriteFile(lockFile, []byte(bundleYaml), 0777)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	imgRetriever := &fakeImageRetriever{
		ImageReturns: []imageRetrieverImageReturns{{img: fakeImage(nil), err: nil}, {img: fakeBundleOCIImage(t), err: nil}},
	}

	_, err = lf.CollectImageLockURLs(lockFile, imgRetriever)
	if err == nil {
		t.Fatalf("unexpected success: %s", err)
	}

	if !strings.Contains(err.Error(), "expected not to contain bundle reference") {
		t.Fatalf("Expected a different error: %s", err.Error())
	}
}

func TestCollectImageLockURLs_Success_ReturnsListOfImgReferencesInImageLockFile(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "bundle-tests")
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	err := os.MkdirAll(testDir, 0700)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	bundleYaml := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: some/image@sha256:ecde7be1a7f238ea1721f8f8ba5dddae07a53e35eb1bb1f242e97db4aa2bc890
  - image: localhost:5000/some/other_image@sha256:e0c01e71eb67453430e43947b628a40d2cca58ed7154bc33339fc3b0f3189671
`
	err = ioutil.WriteFile(lockFile, []byte(bundleYaml), 0777)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	imgRetriever := &fakeImageRetriever{
		ImageReturns: []imageRetrieverImageReturns{{img: fakeImage(nil), err: nil}},
	}

	imgsInLock, err := lf.CollectImageLockURLs(lockFile, imgRetriever)
	if err != nil {
		t.Fatalf("unexpected success: %s", err)
	}

	if len(imgsInLock) != 2 {
		t.Fatalf("expected to have 2 images but got %d", len(imgsInLock))
	}

	if imgsInLock[0].Name() != "index.docker.io/some/image@sha256:ecde7be1a7f238ea1721f8f8ba5dddae07a53e35eb1bb1f242e97db4aa2bc890" {
		t.Fatalf("expected image to be some/image but got %s", imgsInLock[0].Name())
	}

	if imgsInLock[1].Name() != "localhost:5000/some/other_image@sha256:e0c01e71eb67453430e43947b628a40d2cca58ed7154bc33339fc3b0f3189671" {
		t.Fatalf("expected image to be some/other_image but got %s", imgsInLock[1].Name())
	}
}

func fakeImage(labels map[string]string) regv1.Image {
	img := &fake.FakeImage{}
	img.ConfigFileReturns(&regv1.ConfigFile{
		Config: regv1.Config{
			Labels: labels,
		},
	}, nil)

	return img
}

func fakeBundleOCIImage(t *testing.T) regv1.Image {
	img, err := random.Image(0, 0)
	if err != nil {
		t.Fatalf("unable to create image: %s", err)
	}

	img, err = mutate.ConfigFile(img, &regv1.ConfigFile{
		Config: regv1.Config{
			Labels: map[string]string{image.BundleConfigLabel: "i-am-a-bundle"},
		},
	})
	if err != nil {
		t.Fatalf("unable to update the labels: %s", err)
	}

	buf, err := ioutil.ReadFile(filepath.Join("testdata", "bundle_layer.tar"))
	if err != nil {
		t.Fatalf("unable to read layer file: %s", err)
	}

	layerReader, err := tarball.LayerFromReader(bytes.NewBuffer(buf))
	if err != nil {
		t.Fatalf("unable to create layer: %s", err)
	}

	img, err = mutate.AppendLayers(img, layerReader)
	if err != nil {
		t.Fatalf("unable to append the layer: %s", err)
	}

	return img
}

type imageRetrieverImageReturns struct {
	img regv1.Image
	err error
}

type fakeImageRetriever struct {
	ImageReturns           []imageRetrieverImageReturns
	imageReturnsCallNumber int
}

func (f *fakeImageRetriever) Image(ref regname.Reference) (regv1.Image, error) {
	nextResult := f.imageReturnsCallNumber % len(f.ImageReturns)
	f.imageReturnsCallNumber++
	result := f.ImageReturns[nextResult]
	return result.img, result.err
}
