// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"io/ioutil"
	"math/rand"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func CompareFiles(t *testing.T, path1, path2 string) {
	t.Helper()
	path1Bs, err := ioutil.ReadFile(path1)
	require.NoError(t, err, "reading path1")

	path2Bs, err := ioutil.ReadFile(path2)
	require.NoError(t, err, "reading path2")

	require.Equal(t, string(path2Bs), string(path1Bs))
}

const BundleYAML = `---
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
const ImagesYAML = `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
`
const ImageFile = "images.yml"
const BundleFile = "bundle.yml"

func ExtractDigest(t *testing.T, out string) string {
	t.Helper()
	match := regexp.MustCompile("@(sha256:[0123456789abcdef]{64})").FindStringSubmatch(out)
	require.Len(t, match, 2)
	return match[1]
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
