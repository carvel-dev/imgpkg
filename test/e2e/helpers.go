// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"github.com/k14s/difflib"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func compareFiles(path1, path2 string, t *testing.T) {
	t.Helper()
	path1Bs, err := ioutil.ReadFile(path1)
	if err != nil {
		t.Fatalf("reading path1: %s", err)
	}

	path2Bs, err := ioutil.ReadFile(path2)
	if err != nil {
		t.Fatalf("reading path2: %s", err)
	}

	if string(path1Bs) != string(path2Bs) {
		t.Fatalf("Expected contents to match for %s vs %s\nDiff: %s", path1, path2, diffText(string(path1Bs), string(path2Bs)))
	}
}

func diffText(left, right string) string {
	var sb strings.Builder

	recs := difflib.Diff(strings.Split(right, "\n"), strings.Split(left, "\n"))

	for _, diff := range recs {
		var mark string

		switch diff.Delta {
		case difflib.RightOnly:
			mark = " + |"
		case difflib.LeftOnly:
			mark = " - |"
		case difflib.Common:
			mark = "   |"
		}

		// make sure to have line numbers to make sure diff is truly unique
		sb.WriteString(fmt.Sprintf("%3d,%3d%s%s\n", diff.LineLeft, diff.LineRight, mark, diff.Payload))
	}

	return sb.String()
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
	t.Helper()
	match := regexp.MustCompile("@(sha256:[0123456789abcdef]{64})").FindStringSubmatch(out)
	if len(match) != 2 {
		t.Fatalf("Expected to find digest in output '%s'", out)
	}
	return match[1]
}
