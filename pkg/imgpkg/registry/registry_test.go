// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
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

		subject, err := registry.NewSimpleRegistry(registry.Opts{})
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

		subject, err := registry.NewSimpleRegistry(registry.Opts{})
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
		subject, err := registry.NewSimpleRegistry(registry.Opts{})
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

func TestInsecureRegistryFlag(t *testing.T) {
	tests := []struct {
		fName string
		exec  func(t *testing.T, r registry.Registry) error
	}{
		{
			fName: "Get",
			exec: func(t *testing.T, r registry.Registry) error {
				img, err := name.ParseReference("my.registry.io/some/image")
				require.NoError(t, err)
				_, err = r.Get(img)
				return err
			},
		},
		{
			fName: "Digest",
			exec: func(t *testing.T, r registry.Registry) error {
				img, err := name.ParseReference("my.registry.io/some/image")
				require.NoError(t, err)
				_, err = r.Digest(img)
				return err
			},
		},
		{
			fName: "Image",
			exec: func(t *testing.T, r registry.Registry) error {
				img, err := name.ParseReference("my.registry.io/some/image")
				require.NoError(t, err)
				_, err = r.Image(img)
				return err
			},
		},
		{
			fName: "Index",
			exec: func(t *testing.T, r registry.Registry) error {
				img, err := name.ParseReference("my.registry.io/some/image")
				require.NoError(t, err)
				_, err = r.Index(img)
				return err
			},
		},
		{
			fName: "ListTags",
			exec: func(t *testing.T, r registry.Registry) error {
				img, err := name.ParseReference("my.registry.io/some/image")
				require.NoError(t, err)
				_, err = r.ListTags(img.Context())
				return err
			},
		},
		{
			fName: "MultiWrite",
			exec: func(t *testing.T, r registry.Registry) error {
				img, err := name.ParseReference("my.registry.io/some/image")
				require.NoError(t, err)
				return r.MultiWrite(map[name.Reference]regremote.Taggable{img: nil}, 1, nil)
			},
		},
		{
			fName: "WriteImage",
			exec: func(t *testing.T, r registry.Registry) error {
				img, err := name.ParseReference("my.registry.io/some/image")
				require.NoError(t, err)
				return r.WriteImage(img, nil, nil)
			},
		},
		{
			fName: "WriteIndex",
			exec: func(t *testing.T, r registry.Registry) error {
				img, err := name.ParseReference("my.registry.io/some/image")
				require.NoError(t, err)
				return r.WriteIndex(img, nil)
			},
		},
		{
			fName: "WriteTag",
			exec: func(t *testing.T, r registry.Registry) error {
				img, err := name.NewTag("my.registry.io/some/image:tag")
				require.NoError(t, err)
				return r.WriteTag(img, nil)
			},
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("when insecure-registry flag is present, %s uses HTTP", test.fName), func(t *testing.T) {
			reqNumber := 0
			rTripper := &notFoundRoundTripper{
				do: func(request *http.Request) (*http.Response, error) {
					defer func() { reqNumber++ }()
					if reqNumber == 1 {
						assert.Equal(t, "http", request.URL.Scheme)
						return &http.Response{
							Status:     "Not Found",
							StatusCode: http.StatusNotFound,
						}, nil
					}

					assert.Equal(t, "https", request.URL.Scheme)
					return &http.Response{
						Status:     "Not Found",
						StatusCode: http.StatusNotFound,
					}, errors.New("not found")

				},
			}
			subject, err := registry.NewSimpleRegistryWithTransport(registry.Opts{
				Insecure: true,
			}, rTripper)
			require.NoError(t, err)

			err = test.exec(t, subject)

			assert.Equal(t, 2, reqNumber, "Should call the registry twice, once with https and a second one with http")
			require.ErrorContains(t, err, "404 Not Found")
		})

		t.Run(fmt.Sprintf("when insecure-registry flag is set to false, %s uses HTTPS", test.fName), func(t *testing.T) {
			reqNumber := 0
			rTripper := &notFoundRoundTripper{
				do: func(request *http.Request) (*http.Response, error) {
					defer func() { reqNumber++ }()
					assert.Equal(t, "https", request.URL.Scheme)
					return &http.Response{
						Status:     "Not Found",
						StatusCode: http.StatusNotFound,
					}, errors.New("not found")
				},
			}
			subject, err := registry.NewSimpleRegistryWithTransport(registry.Opts{
				Insecure: false,
			}, rTripper)
			require.NoError(t, err)

			err = test.exec(t, subject)

			assert.Equal(t, 1, reqNumber, "Should call the registry once, once with https")
			require.ErrorContains(t, err, "not found")
		})
	}
}

type notFoundRoundTripper struct {
	do func(request *http.Request) (*http.Response, error)
}

func (n *notFoundRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return n.do(request)
}
