// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestExpectedRegistry tests an expected
// value of DOCKERHUB_PROXY
func TestExpectedRegistry(t *testing.T) {

	v, isSet := os.LookupEnv("DOCKERHUB_PROXY")
	if isSet {
		defer os.Setenv("DOCKERHUB_PROXY", v)
	}

	os.Unsetenv("DOCKERHUB_PROXY")
	assert.Equal(t, "index.docker.io", GetDockerHubRegistry())

	os.Setenv("DOCKERHUB_PROXY", "my-dockerhub-proxy.tld/dockerhub-proxy")
	assert.Equal(t, "my-dockerhub-proxy.tld/dockerhub-proxy", GetDockerHubRegistry())
	os.Unsetenv("DOCKERHUB_PROXY")

}

// TestExpectedImgRef tests an expected
// value of image reference
func TestExpectedImgRef(t *testing.T) {

	v, isSet := os.LookupEnv("DOCKERHUB_PROXY")
	if isSet {
		defer os.Setenv("DOCKERHUB_PROXY", v)
	}

	os.Unsetenv("DOCKERHUB_PROXY")
	assert.Equal(t,
		"index.docker.io/library/hello-world@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6",
		CompleteImageRef("library/hello-world@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6"))

	os.Setenv("DOCKERHUB_PROXY", "my-dockerhub-proxy.tld/dockerhub-proxy")
	assert.Equal(t,
		"my-dockerhub-proxy.tld/dockerhub-proxy/library/hello-world@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6",
		CompleteImageRef("library/hello-world@sha256:ebf526c198a14fa138634b9746c50ec38077ec9b3986227e79eb837d26f59dc6"))
	os.Unsetenv("DOCKERHUB_PROXY")
}
