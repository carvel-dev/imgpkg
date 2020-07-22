package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func TestBundlePush(t *testing.T) {
	// Do some setup
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}}
	assetsDir := filepath.Join("assets", "bundle-dir")

	// push the bundle in the assets dir
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsDir})

	// Validate bundle annotation is present
	ref, _ := name.NewTag(env.Image, name.WeakValidation)
	image, err := remote.Image(ref)
	if err != nil {
		t.Fatalf("Error getting remote image in test: %s", err)
	}

	manifestBs, err := image.RawManifest()
	if err != nil {
		t.Fatalf("Error getting manifest in test: %s", err)
	}

	var manifest v1.Manifest
	err = json.Unmarshal(manifestBs, &manifest)
	if err != nil {
		t.Fatalf("Error unmarshaling manifest in test: %s", err)
	}

	if val, found := manifest.Annotations["io.k14s.imgpkg.bundle"]; !found || val != "true" {
		t.Fatalf("Expected manifest to contain bundle annotation, instead had: %v", manifest.Annotations)
	}
}

func TestBundlePushLockOutput(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}}
	assetsDir := filepath.Join("assets", "bundle-dir")
	bundleLock := filepath.Join(os.TempDir(), "imgpkg-bundle-lock-test.yml")
	os.RemoveAll(bundleLock)

	defer os.RemoveAll(bundleLock)

	// push the bundle in the assets dir
	imgpkg.Run([]string{"push", "-b", env.Image, "-f", assetsDir, "--lock-output", bundleLock})

	bundleBs, err := ioutil.ReadFile(bundleLock)
	if err != nil {
		t.Fatalf("Could not read bundle lock file in test: %s", err)
	}

	expectedYml := fmt.Sprintf(`---
apiVersion: imgpkg.k14s.io/v1alpha1
kind: BundleLock
spec:
  image:
    url: %s@sha256:ee306375f86619455da201fca6ddc81765c639e03891af0110a84fc5aa649a51
    tag: latest
`, env.Image)

	if string(bundleBs) != expectedYml {
		t.Fatalf("Expected BundleLock to match:\n\n %s\n\n, got:\n\n %s\n", expectedYml, string(bundleBs))
	}
}
