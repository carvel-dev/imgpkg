package cmd_test

import (
	"strings"
	"testing"

	"github.com/k14s/imgpkg/pkg/imgpkg/cmd"
	"gopkg.in/yaml.v2"
)

func TestImageLockUnmarshalError(t *testing.T) {
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
