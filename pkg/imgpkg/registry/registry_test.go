// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
)

func TestRegistry_Digest(t *testing.T) {
	t.Run("when can find image it returns the digest", func(t *testing.T) {
		expectedDigest := "sha256:477c34d98f9e090a4441cf82d2f1f03e64c8eb730e8c1ef39a8595e685d4df65"
		server := createServer(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Docker-Content-Digest", expectedDigest)
		})
		defer server.Close()
		u, err := url.Parse(server.URL)
		require.NoError(t, err)

		subject, err := registry.NewRegistry(registry.Opts{})
		require.NoError(t, err)

		imgRef, err := name.ParseReference(fmt.Sprintf("%s/repo:latest", u.Host))
		require.NoError(t, err)
		digest, err := subject.Digest(imgRef)
		require.NoError(t, err)
		require.Equal(t, expectedDigest, digest.String())
	})

	t.Run("when HEAD Request fails, it executes a GET and returns the expected digest", func(t *testing.T) {
		expectedDigest := "sha256:477c34d98f9e090a4441cf82d2f1f03e64c8eb730e8c1ef39a8595e685d4df65"
		getCalled := false

		server := createServer(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				getCalled = true
				w.Header().Set("Docker-Content-Digest", expectedDigest)
			}
		})
		defer server.Close()
		u, err := url.Parse(server.URL)
		require.NoError(t, err)

		subject, err := registry.NewRegistry(registry.Opts{})
		require.NoError(t, err)
		imgRef, err := name.ParseReference(fmt.Sprintf("%s/repo:latest", u.Host))
		require.NoError(t, err)
		digest, err := subject.Digest(imgRef)
		require.NoError(t, err)
		require.Equal(t, expectedDigest, digest.String())
		require.True(t, getCalled)
	})
}

func createServer(handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	response := []byte("doesn't matter")
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", string(types.DockerManifestSchema2))
		handler(w, r)
		w.Write(response)
	}))
}

func TestRegistry_Get(t *testing.T) {
	t.Run("when Ref includes protocol it errors", func(t *testing.T) {
		subject, err := registry.NewRegistry(registry.Opts{})
		require.NoError(t, err)

		ref, err := name.NewTag("https://docker.whatever/whoever/etc")
		require.NoError(t, err)
		_, err = subject.Get(ref)
		assert.Error(t, err)
		assert.True(t,
			strings.Contains(err.Error(), "should not include https://"),
			fmt.Sprintf("error returned from Get was expected to be about protocol but was: %v", err))

		// same as above but with no 's' after http
		ref, err = name.NewTag("http://docker.whatever/whoever/etc")
		require.NoError(t, err)
		_, err = subject.Get(ref)
		assert.Error(t, err)
		assert.True(t,
			strings.Contains(err.Error(), "should not include http://"),
			fmt.Sprintf("error returned from Get was expected to be about protocol but was: %v", err))
	})

}
