// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/registry"
	"github.com/stretchr/testify/require"
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
