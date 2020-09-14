package e2e

import (
	"bytes"
	"fmt"
	"github.com/k14s/imgpkg/pkg/imgpkg/cmd"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func TestCopyBundleLockInputToRepo(t *testing.T) {
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
apiVersion: imgpkg.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: image
    url: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	// create bundle that refs image with --lock-ouput and a random tag based on time
	imgpkg.Run([]string{"push", "-b", fmt.Sprintf("%s:%v", env.Image, time.Now().UnixNano()), "-f", assetsPath, "--lock-output", lockFile})
	bundleLockYml, err := cmd.ReadBundleLockFile(lockFile)
	if err != nil {
		t.Fatalf("failed to read bundlelock file: %v", err)
	}
	bundleDigest := fmt.Sprintf("@%s", extractDigest(bundleLockYml.Spec.Image.DigestRef, t))
	bundleTag := fmt.Sprintf(":%s", bundleLockYml.Spec.Image.OriginalTag)

	// copy via output file
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo})

	// check if bundle and referenced images are present in dst repo
	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest, env.RelocationRepo + bundleTag}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}

}

func TestCopyImageLockInputToRepo(t *testing.T) {
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

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigest

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: image
    url: %s
`, imageDigestRef)

	err = ioutil.WriteFile(lockFile, []byte(imgsYml), 0700)
	if err != nil {
		t.Fatalf("failed to create images.lock file: %v", err)
	}

	// copy via output file
	imgpkg.Run([]string{"copy", "--lock", lockFile, "--to-repo", env.RelocationRepo})

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigest}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyBundleWithCollocatedReferencedImagesToRepo(t *testing.T) {
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
apiVersion: imgpkg.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: image
    url: index.docker.io/k8slt/imgpkg-test%s
`, imageDigest)

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

	// copy via created ref
	imgpkg.Run([]string{"copy", "--bundle", fmt.Sprintf("%s%s", env.Image, bundleTag), "--to-repo", env.RelocationRepo})

	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleTag, env.RelocationRepo + bundleDigest}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyBundleWithNonCollocatedReferencedImagesToRepo(t *testing.T) {
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

	out := imgpkg.Run([]string{"push", "--tty", "-i", "index.docker.io/k8slt/test", "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	// fallback url for image, image intentionally does not exist in this repo
	imageDigestRef := "index.docker.io/k8slt/test" + imageDigest

	imgsYml := fmt.Sprintf(`---
apiVersion: imgpkg.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: image
    url: %s
`, imageDigestRef)

	// create a bundle with ref to image
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	// create bundle that refs image and a random tag based on time
	out = imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", assetsPath})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	bundleDigestRef := env.Image + bundleDigest

	// copy via created ref
	imgpkg.Run([]string{"copy", "--bundle", bundleDigestRef, "--to-repo", env.RelocationRepo})

	refs := []string{env.RelocationRepo + imageDigest, env.RelocationRepo + bundleDigest}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyImageInputToRepo(t *testing.T) {
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

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigestTag := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigestTag

	// copy via create ref
	imgpkg.Run([]string{"copy", "--image", imageDigestRef, "--to-repo", env.RelocationRepo})

	// check if image is present in dst repo
	refs := []string{env.RelocationRepo + imageDigestTag}
	if err := validateImagePresence(refs); err != nil {
		t.Fatalf("could not validate image presence: %v", err)
	}
}

func TestCopyBundleLockInputToTar(t *testing.T) {
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
apiVersion: imgpkg.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: image
    url: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	// create bundle that refs image with --lock-ouput
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsPath, "--lock-output", lockFile})
	bundleLockYml, err := cmd.ReadBundleLockFile(lockFile)
	if err != nil {
		t.Fatalf("failed to read bundlelock file: %v", err)
	}
	bundleDigestRef := fmt.Sprintf("%s@%s", env.Image, extractDigest(bundleLockYml.Spec.Image.DigestRef, t))

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
}

func TestCopyImageLockInputToTar(t *testing.T) {
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
apiVersion: imgpkg.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: image
    url: %s
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
}

func TestCopyBundleInputToTarThenToRepo(t *testing.T) {
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
apiVersion: imgpkg.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: image
    url: %s
`, imageDigestRef)

	// create a bundle with ref to generic
	imgpkgDir, err := createBundleDir(assetsPath, bundleYAML, imgsYml)
	if err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}
	defer os.RemoveAll(imgpkgDir)

	// create bundle that refs image
	out = imgpkg.Run([]string{"push", "--tty", "-b", env.Image, "-f", assetsPath})
	bundleDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	bundleDigestRef := env.Image + bundleDigest

	// copy to a tar
	imgpkg.Run([]string{"copy", "-b", bundleDigestRef, "--to-tar", tarFilePath})

	// validate tar contains bundle and image
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

	// copy from tar to repo
	imgpkg.Run([]string{"copy", "--from-tar", tarFilePath, "--to-repo", env.RelocationRepo})

	// validate bundle and image were relocated
	relocatedBundleRef := env.RelocationRepo + bundleDigest
	relocatedImageRef := env.RelocationRepo + imageDigest

	if err := validateImagePresence([]string{relocatedBundleRef, relocatedImageRef}); err != nil {
		t.Fatalf("Failed to locate digest in relocationRepo: %v", err)
	}
}

func TestCopyImageInputToTar(t *testing.T) {
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

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image, "-f", assetsPath})
	imageDigest := fmt.Sprintf("@%s", extractDigest(out, t))
	imageDigestRef := env.Image + imageDigest

	// copy to tar
	imgpkg.Run([]string{"copy", "-i", imageDigestRef, "--to-tar", tarFilePath})

	// validate image was relocated
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
apiVersion: imgpkg.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: image
    url: %s
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
		if _, err := remote.Image(ref); err != nil {
			return fmt.Errorf("validating image %s: %v", refString, err)
		}
	}
	return nil
}
