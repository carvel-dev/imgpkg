// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func compareFiles(path1, path2 string, t *testing.T) {
	path1Bs, err := ioutil.ReadFile(path1)
	if err != nil {
		t.Fatalf("reading path1: %s", err)
	}

	path2Bs, err := ioutil.ReadFile(path2)
	if err != nil {
		t.Fatalf("reading path2: %s", err)
	}

	if string(path1Bs) != string(path2Bs) {
		t.Fatalf("Expected contents to match for %s vs %s", path1, path2)
	}
}

const bundleYAML = `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: Bundle
metadata:
  name: my-app
authors:
- name: blah
  email: blah@blah.com
websites:
- url: blah.com
`
const imagesYAML = `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images: []
`
const imageFile = "images.yml"
const bundleFile = "bundle.yml"

func createBundleDir(dir, bYml, iYml string) (string, error) {
	imgpkgDir := filepath.Join(dir, ".imgpkg")
	err := os.Mkdir(imgpkgDir, 0700)
	if err != nil {
		return "", err
	}

	fileContents := map[string]string{
		bundleFile: bYml,
		imageFile:  iYml,
	}
	for filename, contents := range fileContents {
		err = ioutil.WriteFile(filepath.Join(imgpkgDir, filename), []byte(contents), 0600)
		if err != nil {
			return imgpkgDir, err
		}
	}
	return imgpkgDir, nil
}

func extractDigest(out string, t *testing.T) string {
	match := regexp.MustCompile("@(sha256:[0123456789abcdef]{64})").FindStringSubmatch(out)
	if len(match) != 2 {
		t.Fatalf("Expected to find digest in output '%s'", out)
	}
	return match[1]
}
