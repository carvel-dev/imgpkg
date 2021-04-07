// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package registry_test

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/registry"
	"github.com/stretchr/testify/assert"
)

func TestAnonAuthWhenNoEnvVarsProvided(t *testing.T) {
	keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return nil })
	resource, err := name.NewRepository("some.fake.registry/imgpkg_test")
	assert.NoError(t, err)

	auth, err := keychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, authn.Anonymous, auth)
}

func TestEnvAuthWhenEnvVarsProvided(t *testing.T) {
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
}

func TestEnvAuthWhenEnvVarsProvidedWithMultipleRegistries(t *testing.T) {
	envVars := []string{
		"IMGPKG_REGISTRY_USERNAME_0=user_0",
		"IMGPKG_REGISTRY_PASSWORD_0=pass_0",
		"IMGPKG_REGISTRY_HOSTNAME_0=localhost:0000",

		"IMGPKG_REGISTRY_USERNAME_1=user_1",
		"IMGPKG_REGISTRY_PASSWORD_1=pass_1",
		"IMGPKG_REGISTRY_HOSTNAME_1=localhost:1111",
	}

	keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return envVars })
	resource, err := name.NewRepository("localhost:1111/imgpkg_test")
	assert.NoError(t, err)

	auth, err := keychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, authn.FromConfig(authn.AuthConfig{
		Username: "user_1",
		Password: "pass_1",
	}), auth)
}

func TestRegistryToken(t *testing.T) {
	envVars := []string{
		"IMGPKG_REGISTRY_REGISTRY_TOKEN=reg_token",
		"IMGPKG_REGISTRY_HOSTNAME=localhost:9999",
	}

	keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return envVars })
	resource, err := name.NewRepository("localhost:9999/imgpkg_test")
	assert.NoError(t, err)

	auth, err := keychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, authn.FromConfig(authn.AuthConfig{
		RegistryToken: "reg_token",
	}), auth)
}

func TestIdentityToken(t *testing.T) {
	envVars := []string{
		"IMGPKG_REGISTRY_IDENTITY_TOKEN=id_token",
		"IMGPKG_REGISTRY_HOSTNAME=localhost:9999",
	}

	keychain := registry.Keychain(registry.KeychainOpts{}, func() []string { return envVars })
	resource, err := name.NewRepository("localhost:9999/imgpkg_test")
	assert.NoError(t, err)

	auth, err := keychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, authn.FromConfig(authn.AuthConfig{
		IdentityToken: "id_token",
	}), auth)
}

func TestAuthFromFlagHasPrecedenceOverEnvVars(t *testing.T) {
	envVars := []string{
		"IMGPKG_REGISTRY_USERNAME=env_user",
		"IMGPKG_REGISTRY_PASSWORD=env_pass",
		"IMGPKG_REGISTRY_HOSTNAME=localhost:9999",
	}

	keychain := registry.Keychain(registry.KeychainOpts{Username: "flag_user", Password: "flag_pass"}, func() []string { return envVars })
	resource, err := name.NewRepository("localhost:9999/imgpkg_test")
	assert.NoError(t, err)

	auth, err := keychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, &authn.Basic{
		Username: "flag_user",
		Password: "flag_pass",
	}, auth)
}
