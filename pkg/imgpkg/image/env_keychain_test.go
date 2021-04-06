// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package image

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
)

const globalTestPrefix string = "TEST_IMGPKG_REGISTRY"

func TestAnonAuthWhenNoEnvVarsProvided(t *testing.T) {
	envKeychain := NewEnvKeychain(globalTestPrefix)
	resource, err := name.NewRepository("imgpkg_test")
	assert.NoError(t, err)

	auth, err := envKeychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, authn.Anonymous, auth)
}

func TestEnvAuthWhenEnvVarsProvided(t *testing.T) {
	setTestEnv(t, "USERNAME", "user")
	setTestEnv(t, "PASSWORD", "pass")
	setTestEnv(t, "HOSTNAME", "localhost:9999")

	defer unsetTestEnv(t)

	envKeychain := NewEnvKeychain(globalTestPrefix)
	resource, err := name.NewRepository("localhost:9999/imgpkg_test")
	assert.NoError(t, err)

	auth, err := envKeychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, authn.FromConfig(authn.AuthConfig{
		Username: "user",
		Password: "pass",
	}), auth)
}

func TestEnvAuthWhenEnvVarsProvidedWithMultipleRegistries(t *testing.T) {
	setTestEnv(t, "USERNAME_0", "user_0")
	setTestEnv(t, "PASSWORD_0", "pass_0")
	setTestEnv(t, "HOSTNAME_0", "localhost:0000")

	setTestEnv(t, "USERNAME_1", "user_1")
	setTestEnv(t, "PASSWORD_1", "pass_1")
	setTestEnv(t, "HOSTNAME_1", "localhost:1111")

	defer unsetTestEnv(t)

	envKeychain := NewEnvKeychain(globalTestPrefix)
	resource, err := name.NewRepository("localhost:1111/imgpkg_test")
	assert.NoError(t, err)

	auth, err := envKeychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, authn.FromConfig(authn.AuthConfig{
		Username: "user_1",
		Password: "pass_1",
	}), auth)
}

func TestRegistryToken(t *testing.T) {
	setTestEnv(t, "HOSTNAME", "localhost:1111")
	setTestEnv(t, "REGISTRY_TOKEN", "TOKEN")

	defer unsetTestEnv(t)

	envKeychain := NewEnvKeychain(globalTestPrefix)
	resource, err := name.NewRepository("localhost:1111/imgpkg_test")
	assert.NoError(t, err)

	auth, err := envKeychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, authn.FromConfig(authn.AuthConfig{
		RegistryToken: "TOKEN",
	}), auth)
}

func TestIdentityToken(t *testing.T) {
	setTestEnv(t, "HOSTNAME", "localhost:1111")
	setTestEnv(t, "IDENTITY_TOKEN", "ID_TOKEN")

	defer unsetTestEnv(t)

	envKeychain := NewEnvKeychain(globalTestPrefix)
	resource, err := name.NewRepository("localhost:1111/imgpkg_test")
	assert.NoError(t, err)

	auth, err := envKeychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, authn.FromConfig(authn.AuthConfig{
		IdentityToken: "ID_TOKEN",
	}), auth)
}

func setTestEnv(t *testing.T, key string, value string) {
	assert.NoError(t, os.Setenv(globalTestPrefix+"_"+key, value))
}

func unsetTestEnv(t *testing.T) {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, globalTestPrefix) {
			assert.NoError(t, os.Unsetenv(strings.Split(env, "=")[0]))
		}
	}
}
