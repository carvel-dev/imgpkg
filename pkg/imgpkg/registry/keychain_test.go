// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package registry_test

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/registry"
	"github.com/k14s/imgpkg/pkg/imgpkg/registry/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	credentialprovider "github.com/vdemeester/k8s-pkg-credentialprovider"
	"github.com/vdemeester/k8s-pkg-credentialprovider/gcp"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/legacy-cloud-providers/gce/gcpcredential"
)

var gcpRegistryURL string
var gcpRegistryUsername string
var gcpRegistryPassword string

func TestMain(m *testing.M) {
	var server *httptest.Server
	gcpRegistryURL, server = registerGCPProvider()
	defer server.Close()

	os.Exit(m.Run())
}

func TestAuthProvidedViaGCP(t *testing.T) {
	t.Run("Should auth via GCP metadata service", func(t *testing.T) {
		keychain, err := registry.Keychain(auth.KeychainOpts{}, func() []string { return nil })
		require.NoError(t, err)

		resource, err := name.NewRepository(fmt.Sprintf("%s/imgpkg_test", gcpRegistryURL))
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		authorization, err := auth.Authorization()
		assert.NoError(t, err)
		assert.Equal(t, "foo", authorization.Username)
		assert.Equal(t, "bar", authorization.Password)
	})

	t.Run("Should be able to disable Iaas providers via env", func(t *testing.T) {
		envVars := []string{
			"IMGPKG_ENABLE_IAAS_AUTH=false",
		}

		keychain, err := registry.Keychain(auth.KeychainOpts{}, func() []string { return envVars })
		require.NoError(t, err)

		resource, err := name.NewRepository(fmt.Sprintf("%s/imgpkg_test", gcpRegistryURL))
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		authorization, err := auth.Authorization()
		assert.NoError(t, err)
		assert.Equal(t, "", authorization.Username)
		assert.Equal(t, "", authorization.Password)
	})
}

func TestAuthProvidedViaCLI(t *testing.T) {
	cliOptions := auth.KeychainOpts{}

	t.Run("When username and password is provided", func(t *testing.T) {
		opts := cliOptions
		opts.Username = "user"
		opts.Password = "pass"
		keychain, err := registry.Keychain(opts, func() []string { return nil })
		require.NoError(t, err)

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, &authn.Basic{
			Username: "user",
			Password: "pass"}, auth)
	})

	t.Run("When anon is provided", func(t *testing.T) {
		opts := cliOptions
		opts.Anon = true
		keychain, err := registry.Keychain(opts, func() []string { return nil })
		require.NoError(t, err)

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.Anonymous, auth)
	})

	t.Run("When token is provided", func(t *testing.T) {
		opts := cliOptions
		opts.Token = "TOKEN"

		keychain, err := registry.Keychain(opts, func() []string { return nil })
		require.NoError(t, err)

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, &authn.Bearer{Token: "TOKEN"}, auth)
	})
}

func TestAuthProvidedViaEnvVars(t *testing.T) {
	t.Run("When a single registry credentials is provided", func(t *testing.T) {
		envVars := []string{
			"IMGPKG_REGISTRY_USERNAME=user",
			"IMGPKG_REGISTRY_PASSWORD=pass",
			"IMGPKG_REGISTRY_HOSTNAME=localhost:9999",
		}

		keychain, err := registry.Keychain(auth.KeychainOpts{}, func() []string { return envVars })
		require.NoError(t, err)
		resource, err := name.NewRepository("localhost:9999/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			Username: "user",
			Password: "pass",
		}), auth)
	})

	t.Run("When a single registry access token credentials is provided", func(t *testing.T) {
		envVars := []string{
			"IMGPKG_REGISTRY_REGISTRY_TOKEN=REG_TOKEN",
			"IMGPKG_REGISTRY_HOSTNAME=localhost:9999",
		}

		keychain, err := registry.Keychain(auth.KeychainOpts{}, func() []string { return envVars })
		require.NoError(t, err)
		resource, err := name.NewRepository("localhost:9999/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			RegistryToken: "REG_TOKEN",
		}), auth)
	})

	t.Run("When a single registry identity token credentials is provided", func(t *testing.T) {
		envVars := []string{
			"IMGPKG_REGISTRY_IDENTITY_TOKEN=ID_TOKEN",
			"IMGPKG_REGISTRY_HOSTNAME=localhost:9999",
		}

		keychain, err := registry.Keychain(auth.KeychainOpts{}, func() []string { return envVars })
		require.NoError(t, err)
		resource, err := name.NewRepository("localhost:9999/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			IdentityToken: "ID_TOKEN",
		}), auth)
	})

	t.Run("When multiple registry credentials are provided", func(t *testing.T) {
		envVars := []string{
			"IMGPKG_REGISTRY_USERNAME_0=user0",
			"IMGPKG_REGISTRY_PASSWORD_0=pass0",
			"IMGPKG_REGISTRY_HOSTNAME_0=localhost:9999",

			"IMGPKG_REGISTRY_USERNAME_1=user1",
			"IMGPKG_REGISTRY_PASSWORD_1=pass1",
			"IMGPKG_REGISTRY_HOSTNAME_1=localhost:1111",
		}

		keychain, err := registry.Keychain(auth.KeychainOpts{}, func() []string { return envVars })
		require.NoError(t, err)
		resource, err := name.NewRepository("localhost:1111/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			Username: "user1",
			Password: "pass1",
		}), auth)
	})

	t.Run("Only IMGPKG_REGISTRY env variables are used", func(t *testing.T) {
		envVars := []string{
			"IMGPKG_REGISTRY_USERNAME=user",
			"IMGPKG_REGISTRY_PASSWORD=pass",
			"IMGPKG_REGISTRY_HOSTNAME=localhost:9999",

			"SOMETHING_REGISTRY_USERNAME=wrong-user",
			"SOMETHING_REGISTRY_PASSWORD=wrong-pass",
			"SOMETHING_REGISTRY_HOSTNAME=localhost:9999",
		}

		keychain, err := registry.Keychain(auth.KeychainOpts{}, func() []string { return envVars })
		require.NoError(t, err)
		resource, err := name.NewRepository("localhost:9999/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			Username: "user",
			Password: "pass",
		}), auth)
	})

	for i, validHostname := range []string{
		"localhost:9999",
		"http://localhost:9999",
		"https://localhost:9999",
		"localhost:9999/v1/",
		"localhost:9999/v2/",
	} {
		t.Run(fmt.Sprintf("IMGPKG_HOSTNAME %d", i), func(t *testing.T) {
			envVars := []string{
				"IMGPKG_REGISTRY_USERNAME=user",
				"IMGPKG_REGISTRY_PASSWORD=pass",
				fmt.Sprintf("IMGPKG_REGISTRY_HOSTNAME=%s", validHostname),
			}

			keychain, err := registry.Keychain(auth.KeychainOpts{}, func() []string { return envVars })
			assert.NoError(t, err)

			resource, err := name.NewRepository("localhost:9999/imgpkg_test")
			assert.NoError(t, err)

			auth, err := keychain.Resolve(resource)
			assert.NoError(t, err)

			assert.Equal(t, authn.FromConfig(authn.AuthConfig{
				Username: "user",
				Password: "pass",
			}), auth)
		})
	}

	for i, invalidHostname := range []string{
		"http://[::1]:namedport", // rfc3986 3.2.3
		"http://[%10::1]",        // no %xx escapes in IP address
		"http://%41:8080/",       // not allowed: % encoding only for non-ASCII
	} {
		t.Run(fmt.Sprintf("IMGPKG_HOSTNAME %d", i), func(t *testing.T) {
			envVars := []string{
				"IMGPKG_REGISTRY_USERNAME=user",
				"IMGPKG_REGISTRY_PASSWORD=pass",
				fmt.Sprintf("IMGPKG_REGISTRY_HOSTNAME=%s", invalidHostname),
			}

			keychain, err := registry.Keychain(auth.KeychainOpts{}, func() []string { return envVars })
			assert.NoError(t, err)

			resource, err := name.NewRepository("localhost:9999/imgpkg_test")
			assert.NoError(t, err)

			_, err = keychain.Resolve(resource)
			assert.Error(t, err)
		})
	}
}

func TestAuthProvidedViaDefaultKeychain(t *testing.T) {
	t.Run("When auth is provided via config.json", func(t *testing.T) {
		tempConfigJSONDir, err := ioutil.TempDir(os.TempDir(), "test-default-keychain")
		assert.NoError(t, err)
		defer os.RemoveAll(tempConfigJSONDir)
		assert.NoError(t, os.Setenv("DOCKER_CONFIG", tempConfigJSONDir))
		defer os.Unsetenv("DOCKER_CONFIG")

		err = ioutil.WriteFile(filepath.Join(tempConfigJSONDir, "config.json"), []byte(`{
  "auths" : {
    "http://localhost:9999/v1/" : {
		"username": "user-config-json",
		"password": "pass-config-json"
    }
  }
}`), os.ModePerm)
		assert.NoError(t, err)

		keychain, err := registry.Keychain(auth.KeychainOpts{}, func() []string { return nil })
		require.NoError(t, err)
		resource, err := name.NewRepository("localhost:9999/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			Username: "user-config-json",
			Password: "pass-config-json",
		}), auth)
	})
}

func TestOrderingOfAuthOpts(t *testing.T) {
	t.Run("When no auth are provided, use anon", func(t *testing.T) {
		cliOptions := auth.KeychainOpts{}

		keychain, err := registry.Keychain(cliOptions, func() []string { return nil })
		require.NoError(t, err)

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.Anonymous, auth)
	})

	t.Run("env creds > iaas", func(t *testing.T) {
		cliOptions := auth.KeychainOpts{}

		envVars := []string{
			"IMGPKG_REGISTRY_USERNAME=user-env",
			"IMGPKG_REGISTRY_PASSWORD=pass-env",
			fmt.Sprintf("IMGPKG_REGISTRY_HOSTNAME=%s", gcpRegistryURL),
		}

		keychain, err := registry.Keychain(cliOptions, func() []string { return envVars })
		require.NoError(t, err)

		resource, err := name.NewRepository(fmt.Sprintf("%s/imgpkg_test", gcpRegistryURL))
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		authorization, err := auth.Authorization()
		assert.NoError(t, err)
		assert.Equal(t, "user-env", authorization.Username)
		assert.Equal(t, "pass-env", authorization.Password)
	})

	t.Run("env creds > cli user/pass", func(t *testing.T) {
		cliOptions := auth.KeychainOpts{
			Username: "user-cli",
			Password: "pass-cli",
		}

		envVars := []string{
			"IMGPKG_REGISTRY_USERNAME=user-env",
			"IMGPKG_REGISTRY_PASSWORD=pass-env",
			"IMGPKG_REGISTRY_HOSTNAME=some.fake.registry",
		}

		keychain, err := registry.Keychain(cliOptions, func() []string { return envVars })
		require.NoError(t, err)

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			Username: "user-env",
			Password: "pass-env",
		}), auth)
	})

	t.Run("env creds > cli anon", func(t *testing.T) {
		cliOptions := auth.KeychainOpts{
			Anon: true,
		}

		envVars := []string{
			"IMGPKG_REGISTRY_USERNAME=user-env",
			"IMGPKG_REGISTRY_PASSWORD=pass-env",
			"IMGPKG_REGISTRY_HOSTNAME=some.fake.registry",
		}

		keychain, err := registry.Keychain(cliOptions, func() []string { return envVars })
		require.NoError(t, err)

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			Username: "user-env",
			Password: "pass-env",
		}), auth)
	})

	t.Run("env creds > config.json", func(t *testing.T) {
		tempConfigJSONDir, err := ioutil.TempDir(os.TempDir(), "test-default-keychain")
		assert.NoError(t, err)
		defer os.RemoveAll(tempConfigJSONDir)
		assert.NoError(t, os.Setenv("DOCKER_CONFIG", tempConfigJSONDir))
		defer os.Unsetenv("DOCKER_CONFIG")

		err = ioutil.WriteFile(filepath.Join(tempConfigJSONDir, "config.json"), []byte(`{
  "auths" : {
    "http://some.fake.registry/v1/" : {
		"username": "user-config-json",
		"password": "pass-config-json"
    }
  }
}`), os.ModePerm)
		assert.NoError(t, err)

		cliOptions := auth.KeychainOpts{}

		envVars := []string{
			"IMGPKG_REGISTRY_USERNAME=user-env",
			"IMGPKG_REGISTRY_PASSWORD=pass-env",
			"IMGPKG_REGISTRY_HOSTNAME=some.fake.registry",
		}

		keychain, err := registry.Keychain(cliOptions, func() []string { return envVars })
		require.NoError(t, err)

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			Username: "user-env",
			Password: "pass-env",
		}), auth)
	})

	t.Run("iaas creds > cli anon", func(t *testing.T) {
		cliOptions := auth.KeychainOpts{
			Anon: true,
		}

		envVars := []string{}

		keychain, err := registry.Keychain(cliOptions, func() []string { return envVars })
		require.NoError(t, err)

		resource, err := name.NewRepository(fmt.Sprintf("%s/imgpkg_test", gcpRegistryURL))
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		authorization, err := auth.Authorization()
		assert.NoError(t, err)
		assert.Equal(t, gcpRegistryUsername, authorization.Username)
		assert.Equal(t, gcpRegistryPassword, authorization.Password)
	})

	t.Run("iaas creds > cli user/pass", func(t *testing.T) {
		cliOptions := auth.KeychainOpts{
			Username: "user-cli",
			Password: "pass-cli",
		}

		envVars := []string{}

		keychain, err := registry.Keychain(cliOptions, func() []string { return envVars })
		require.NoError(t, err)

		resource, err := name.NewRepository(fmt.Sprintf("%s/imgpkg_test", gcpRegistryURL))
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		authorization, err := auth.Authorization()
		assert.NoError(t, err)
		assert.Equal(t, gcpRegistryUsername, authorization.Username)
		assert.Equal(t, gcpRegistryPassword, authorization.Password)
	})

	t.Run("cli anon > config.json", func(t *testing.T) {
		cliOptions := auth.KeychainOpts{
			Anon: true,
		}

		tempConfigJSONDir, err := ioutil.TempDir(os.TempDir(), "test-default-keychain")
		assert.NoError(t, err)
		defer os.RemoveAll(tempConfigJSONDir)
		assert.NoError(t, os.Setenv("DOCKER_CONFIG", tempConfigJSONDir))
		defer os.Unsetenv("DOCKER_CONFIG")

		err = ioutil.WriteFile(filepath.Join(tempConfigJSONDir, "config.json"), []byte(`{
  "auths" : {
    "http://some.fake.registry/v1/" : {
		"username": "user-config-json",
		"password": "pass-config-json"
    }
  }
}`), os.ModePerm)
		assert.NoError(t, err)

		keychain, err := registry.Keychain(cliOptions, func() []string { return nil })
		require.NoError(t, err)

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.Anonymous, auth)
	})
}

func registerGCPProvider() (string, *httptest.Server) {
	registryURL := "imgpkg-testing.kubernetes.carvel"
	email := "foo@bar.baz"
	gcpRegistryUsername = "foo"
	gcpRegistryPassword = "bar"
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", gcpRegistryUsername, gcpRegistryPassword)))
	sampleDockerConfig := fmt.Sprintf(`{
   "https://%s": {
     "email": %q,
     "auth": %q
   }
}`, registryURL, email, auth)
	const probeEndpoint = "/computeMetadata/v1/"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only serve the one metadata key.
		if probeEndpoint == r.URL.Path {
			w.WriteHeader(http.StatusOK)
		} else if strings.HasSuffix(gcpcredential.DockerConfigKey, r.URL.Path) {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, sampleDockerConfig)
		} else {
			http.Error(w, "", http.StatusNotFound)
		}
	}))

	// Make a transport that reroutes all traffic to the example server
	transport := utilnet.SetTransportDefaults(&http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL + req.URL.Path)
		},
	})

	provider := &gcp.DockerConfigKeyProvider{
		MetadataProvider: gcp.MetadataProvider{Client: &http.Client{Transport: transport}},
	}

	credentialprovider.RegisterCredentialProvider("TEST-google-dockercfg-TEST",
		&credentialprovider.CachingDockerConfigProvider{
			Provider: alwaysEnabledProvider{provider},
			Lifetime: 60 * time.Second,
		})

	return registryURL, server
}

type alwaysEnabledProvider struct {
	provider credentialprovider.DockerConfigProvider
}

func (a alwaysEnabledProvider) Enabled() bool {
	return true
}

func (a alwaysEnabledProvider) Provide(image string) credentialprovider.DockerConfig {
	return a.provider.Provide(image)
}
