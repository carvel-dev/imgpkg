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
	"github.com/stretchr/testify/assert"
	credentialprovider "github.com/vdemeester/k8s-pkg-credentialprovider"
	"github.com/vdemeester/k8s-pkg-credentialprovider/gcp"
	utilnet "k8s.io/apimachinery/pkg/util/net"

	"k8s.io/legacy-cloud-providers/gce/gcpcredential"
)

var registryURL string

func TestMain(m *testing.M) {
	var server *httptest.Server
	registryURL, server = registerGCPProvider()
	defer server.Close()

	os.Exit(m.Run())
}

func TestAuthProvidedViaCLI(t *testing.T) {
	cliOptions := registry.KeychainOpts{}

	t.Run("When username and password is provided", func(t *testing.T) {
		opts := cliOptions
		opts.Username = "user"
		opts.Password = "pass"
		keychain := registry.Keychain(opts, func() []string { return nil })

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
		keychain := registry.Keychain(opts, func() []string { return nil })

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.Anonymous, auth)
	})

	t.Run("When token is provided", func(t *testing.T) {
		opts := cliOptions
		opts.Token = "TOKEN"

		keychain := registry.Keychain(opts, func() []string { return nil })

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

		keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return envVars })
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

		keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return envVars })
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

		keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return envVars })
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

		keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return envVars })
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

		keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return envVars })
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

		keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return nil })
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

func TestAuthProvidedViaGCP(t *testing.T) {
	t.Run("Should auth via GCP metadata service", func(t *testing.T) {
		keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return nil })

		resource, err := name.NewRepository(fmt.Sprintf("%s/imgpkg_test", registryURL))
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		authorization, err := auth.Authorization()
		assert.NoError(t, err)
		assert.Equal(t, "foo", authorization.Username)
		assert.Equal(t, "bar", authorization.Password)
	})
}

func TestOrderingOfAuthOpts(t *testing.T) {
	t.Run("When no auth are provided, use anon", func(t *testing.T) {
		cliOptions := registry.KeychainOpts{}

		keychain := registry.Keychain(cliOptions, func() []string { return nil })

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.Anonymous, auth)
	})

	t.Run("env creds > cli user/pass", func(t *testing.T) {
		cliOptions := registry.KeychainOpts{
			Username: "user-cli",
			Password: "pass-cli",
		}

		envVars := []string{
			"IMGPKG_REGISTRY_USERNAME=user-env",
			"IMGPKG_REGISTRY_PASSWORD=pass-env",
			"IMGPKG_REGISTRY_HOSTNAME=some.fake.registry",
		}

		keychain := registry.Keychain(cliOptions, func() []string { return envVars })

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
		cliOptions := registry.KeychainOpts{
			Anon: true,
		}

		envVars := []string{
			"IMGPKG_REGISTRY_USERNAME=user-env",
			"IMGPKG_REGISTRY_PASSWORD=pass-env",
			"IMGPKG_REGISTRY_HOSTNAME=some.fake.registry",
		}

		keychain := registry.Keychain(cliOptions, func() []string { return envVars })

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			Username: "user-env",
			Password: "pass-env",
		}), auth)
	})

	t.Run("cli anon > config.json", func(t *testing.T) {
		cliOptions := registry.KeychainOpts{
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

		keychain := registry.Keychain(cliOptions, func() []string { return nil })

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.Anonymous, auth)
	})

	t.Run("env > config.json", func(t *testing.T) {
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

		cliOptions := registry.KeychainOpts{}

		envVars := []string{
			"IMGPKG_REGISTRY_USERNAME=user-env",
			"IMGPKG_REGISTRY_PASSWORD=pass-env",
			"IMGPKG_REGISTRY_HOSTNAME=some.fake.registry",
		}

		keychain := registry.Keychain(cliOptions, func() []string { return envVars })

		resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
		assert.NoError(t, err)

		auth, err := keychain.Resolve(resource)
		assert.NoError(t, err)

		assert.Equal(t, authn.FromConfig(authn.AuthConfig{
			Username: "user-env",
			Password: "pass-env",
		}), auth)
	})
}

func registerGCPProvider() (string, *httptest.Server) {
	registryURL := "imgpkg-testing.kubernetes.carvel"
	email := "foo@bar.baz"
	username := "foo"
	password := "bar" // Fake value for testing.
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
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
			Provider: alwaysEnabledProvier{provider},
			Lifetime: 60 * time.Second,
		})
	return registryURL, server
}

type alwaysEnabledProvier struct {
	provider credentialprovider.DockerConfigProvider
}

func (a alwaysEnabledProvier) Enabled() bool {
	return true
}

func (a alwaysEnabledProvier) Provide(image string) credentialprovider.DockerConfig {
	return a.provider.Provide(image)
}
