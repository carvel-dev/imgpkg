// Copyright 2024 The Carvel Authors.
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

	"carvel.dev/imgpkg/pkg/imgpkg/registry"
	"github.com/google/go-containerregistry/pkg/name"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
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

func TestRegistry_TransportHeaders(t *testing.T) {
	t.Run("when doing request to registry, imgpkg sends the header imgpkg-session-id", func(t *testing.T) {
		expectedDigest := "sha256:477c34d98f9e090a4441cf82d2f1f03e64c8eb730e8c1ef39a8595e685d4df65"
		server := createServer(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Docker-Content-Digest", expectedDigest)
			require.Equal(t, "673062197574995717", r.Header.Get("imgpkg-session-id"))
		})
		defer server.Close()
		u, err := url.Parse(server.URL)
		require.NoError(t, err)

		subject, err := registry.NewSimpleRegistry(registry.Opts{SessionID: "673062197574995717"})
		require.NoError(t, err)

		imgRef, err := name.ParseReference(fmt.Sprintf("%s/repo:latest", u.Host))
		require.NoError(t, err)
		_, err = subject.Digest(imgRef)
		require.NoError(t, err)
	})

	t.Run("when doing 2 request to registry, the value in the header imgpkg-session-id does not change", func(t *testing.T) {
		expectedDigest := "sha256:477c34d98f9e090a4441cf82d2f1f03e64c8eb730e8c1ef39a8595e685d4df65"
		server := createServer(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Docker-Content-Digest", expectedDigest)
			require.Equal(t, "673062197574995717", r.Header.Get("imgpkg-session-id"))
		})
		defer server.Close()
		u, err := url.Parse(server.URL)
		require.NoError(t, err)

		subject, err := registry.NewSimpleRegistry(registry.Opts{SessionID: "673062197574995717"})
		require.NoError(t, err)

		imgRef, err := name.ParseReference(fmt.Sprintf("%s/repo:latest", u.Host))
		require.NoError(t, err)
		_, err = subject.Digest(imgRef)
		require.NoError(t, err)

		_, err = subject.Digest(imgRef)
		require.NoError(t, err)
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

func TestBasicRegistry(t *testing.T) {
	t.Run("When transport is provided it uses it", func(t *testing.T) {
		rt := &notFoundRoundTripper{do: func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "Not Found",
				StatusCode: http.StatusNotFound,
			}, errors.New("not found")
		}}
		reg, err := registry.NewBasicRegistry(regremote.WithTransport(rt))
		require.NoError(t, err)
		ref, err := name.ParseReference("localhost:1111/does/not/exist")
		require.NoError(t, err)
		_, err = reg.Get(ref)
		require.Error(t, err)
		require.Equal(t, 2, rt.RoundTripNumCalls)
	})

	t.Run("When cloned with CloneWithLogger it still uses initial transport", func(t *testing.T) {
		rt := &notFoundRoundTripper{do: func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "Not Found",
				StatusCode: http.StatusNotFound,
			}, errors.New("not found")
		}}
		reg, err := registry.NewBasicRegistry(regremote.WithTransport(rt))
		require.NoError(t, err)

		cReg := reg.CloneWithLogger(nil)
		ref, err := name.ParseReference("localhost:1111/does/not/exist")
		require.NoError(t, err)
		_, err = cReg.Get(ref)
		require.Error(t, err)
		require.Equal(t, 2, rt.RoundTripNumCalls)
	})

	t.Run("When cloned with CloneWithAuth it still uses initial transport", func(t *testing.T) {
		rt := &notFoundRoundTripper{do: func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "Not Found",
				StatusCode: http.StatusNotFound,
			}, errors.New("not found")
		}}
		reg, err := registry.NewBasicRegistry(regremote.WithTransport(rt))
		require.NoError(t, err)

		tag, err := name.NewTag("localhost:1111/some:tag")
		require.NoError(t, err)

		cReg, err := reg.CloneWithSingleAuth(tag)
		require.NoError(t, err)

		ref, err := name.ParseReference("localhost:1111/does/not/exist")
		require.NoError(t, err)

		_, err = cReg.Get(ref)
		require.Error(t, err)
		require.Equal(t, 2, rt.RoundTripNumCalls)
	})
}

type notFoundRoundTripper struct {
	RoundTripNumCalls int
	do                func(request *http.Request) (*http.Response, error)
}

func (n *notFoundRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	n.RoundTripNumCalls++
	return n.do(request)
}
