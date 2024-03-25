// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/registry"
	"carvel.dev/imgpkg/pkg/imgpkg/registry/auth"
	v1 "carvel.dev/imgpkg/pkg/imgpkg/v1"
	"github.com/stretchr/testify/require"
)

func TestOptsFromEnv(t *testing.T) {
	t.Run("when username is define it does not overwrite it", func(t *testing.T) {
		env := envFake{values: map[string]string{"IMGPKG_USERNAME": "not-used"}}
		opts := registry.Opts{
			Username: "some-username",
		}
		result := v1.OptsFromEnv(opts, env.Value)
		require.Equal(t, opts, result)
	})

	t.Run("when username is NOT define it uses value from the environment", func(t *testing.T) {
		env := envFake{values: map[string]string{"IMGPKG_USERNAME": "should-use"}}
		opts := registry.Opts{}
		result := v1.OptsFromEnv(opts, env.Value)
		require.Equal(t, registry.Opts{Username: "should-use"}, result)
	})

	t.Run("when password is define it does not overwrite it", func(t *testing.T) {
		env := envFake{values: map[string]string{"IMGPKG_PASSWORD": "not-used"}}
		opts := registry.Opts{
			Password: "some-password",
		}
		result := v1.OptsFromEnv(opts, env.Value)
		require.Equal(t, opts, result)
	})

	t.Run("when password is NOT define it uses value from the environment", func(t *testing.T) {
		env := envFake{values: map[string]string{"IMGPKG_PASSWORD": "should-use"}}
		opts := registry.Opts{}
		result := v1.OptsFromEnv(opts, env.Value)
		require.Equal(t, registry.Opts{Password: "should-use"}, result)
	})

	t.Run("when token is define it does not overwrite it", func(t *testing.T) {
		env := envFake{values: map[string]string{"IMGPKG_TOKEN": "not-used"}}
		opts := registry.Opts{
			Token: "some-token",
		}
		result := v1.OptsFromEnv(opts, env.Value)
		require.Equal(t, opts, result)
	})

	t.Run("when token is NOT define it uses value from the environment", func(t *testing.T) {
		env := envFake{values: map[string]string{"IMGPKG_TOKEN": "should-use"}}
		opts := registry.Opts{}
		result := v1.OptsFromEnv(opts, env.Value)
		require.Equal(t, registry.Opts{Token: "should-use"}, result)
	})

	t.Run("when anonymous mode is activated via environment variable it set it", func(t *testing.T) {
		env := envFake{values: map[string]string{"IMGPKG_ANON": "true"}}
		opts := registry.Opts{}
		result := v1.OptsFromEnv(opts, env.Value)
		require.Equal(t, registry.Opts{Anon: true}, result)
	})

	t.Run("when non 'true' string is provided it does not take it into account", func(t *testing.T) {
		env := envFake{values: map[string]string{"IMGPKG_ANON": "not-expected string"}}
		opts := registry.Opts{}
		result := v1.OptsFromEnv(opts, env.Value)
		require.Equal(t, registry.Opts{Anon: false}, result)
	})

	t.Run("when a list of IAAS keychains is provided it adds them as a list", func(t *testing.T) {
		env := envFake{values: map[string]string{"IMGPKG_ACTIVE_KEYCHAINS": "ecr,acr"}}
		opts := registry.Opts{}
		result := v1.OptsFromEnv(opts, env.Value)
		require.Equal(t, registry.Opts{ActiveKeychains: []auth.IAASKeychain{"ecr", "acr"}}, result)
	})

	t.Run("when a single IAAS keychain is provided it adds it to the list", func(t *testing.T) {
		env := envFake{values: map[string]string{"IMGPKG_ACTIVE_KEYCHAINS": "acr"}}
		opts := registry.Opts{}
		result := v1.OptsFromEnv(opts, env.Value)
		require.Equal(t, registry.Opts{ActiveKeychains: []auth.IAASKeychain{"acr"}}, result)
	})
}

type envFake struct {
	values map[string]string
}

func (e envFake) Value(s string) (string, bool) {
	res, found := e.values[s]
	return res, found
}
