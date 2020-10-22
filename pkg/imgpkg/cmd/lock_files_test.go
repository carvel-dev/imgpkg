package cmd_test

import (
	"strings"
	"testing"

	"github.com/k14s/imgpkg/pkg/imgpkg/cmd"
	"gopkg.in/yaml.v2"
)

func TestImageLockNonDigestUnmarshalError(t *testing.T) {
	imageLockYaml := []byte(`apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: image
    url: nginx:v1`)

	var imageLock cmd.ImageLock
	err := yaml.Unmarshal(imageLockYaml, &imageLock)

	if err == nil {
		t.Fatalf("Expected unmarshal to error")
	}

	if msg := err.Error(); !(strings.Contains(msg, "to be in digest form") && strings.Contains(msg, "nginx:v1")) {
		t.Fatalf("Expected unmarshal to fail due to tag ref in lock file")
	}
}

func TestImageLockMissingNameUnmarshalError(t *testing.T) {
	imageLockYaml := []byte(`apiVersion: imgpkg.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: "nginx"
    url: index.docker.io/library/nginx@sha256:36b74457bccb56fbf8b05f79c85569501b721d4db813b684391d63e02287c0b2
  - url: index.docker.io/library/nginx@sha256:36b74457bccb56fbf8b05f79c85569501b721d4db813b684391d63e02287c0b2`)

	var imageLock cmd.ImageLock
	err := yaml.Unmarshal(imageLockYaml, &imageLock)

	if err == nil {
		t.Fatalf("Expected unmarshal to error")
	}

	if msg := err.Error(); !(strings.Contains(msg, "Expected one 'name'") && strings.Contains(msg, "1")) {
		t.Fatalf("Expected unmarshal to fail due to missing name, but got %v", err)
	}
}
