// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvKeychainURLParseErrorDoesNotLeakPassword(t *testing.T) {
	t.Run("malformed URL with password does not leak password in error", func(t *testing.T) {
		// Set up environment with malformed URL containing credentials
		os.Setenv("IMGPKG_REGISTRY_HOSTNAME", "https://user:s3cretPW@host.example:badport/x")
		defer os.Unsetenv("IMGPKG_REGISTRY_HOSTNAME")

		keychain := &EnvKeychain{
			environFunc: os.Environ,
		}

		// Call Resolve which triggers URL parsing
		_, err := keychain.Resolve(&mockResource{})

		// Verify error is returned
		require.Error(t, err)

		// Verify password is NOT in the error message
		require.NotContains(t, err.Error(), "s3cretPW")
		require.NotContains(t, err.Error(), "user:s3cretPW")

		// Verify the error message is safe and informative
		require.Contains(t, err.Error(), "parsing registry hostname")
	})

	t.Run("well-formed URL with valid credentials parses successfully", func(t *testing.T) {
		// Set up environment with valid URL
		os.Setenv("IMGPKG_REGISTRY_HOSTNAME", "https://gcr.io")
		defer os.Unsetenv("IMGPKG_REGISTRY_HOSTNAME")

		keychain := &EnvKeychain{
			environFunc: os.Environ,
		}

		// Should not error
		_, err := keychain.Resolve(&mockResource{})
		require.NoError(t, err)
	})
}

func TestEnvKeychainMalformedCredentialsDoNotLeak(t *testing.T) {
	const secret = "review-secret"

	for _, tc := range []struct {
		name     string
		hostname string
	}{
		{
			name:     "invalid_port",
			hostname: "https://user:" + secret + "@host.example:badport/x",
		},
		{
			name:     "unescaped_slash",
			hostname: "https://user:" + secret + "/suffix@host.example",
		},
		{
			name:     "unescaped_question_mark",
			hostname: "https://user:" + secret + "?suffix@host.example",
		},
		{
			name:     "unescaped_hash",
			hostname: "https://user:" + secret + "#suffix@host.example",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keychain := &EnvKeychain{
				environFunc: func() []string {
					return []string{
						"IMGPKG_REGISTRY_HOSTNAME=" + tc.hostname,
					}
				},
			}

			_, err := keychain.Resolve(&mockResource{})

			require.Error(t, err)
			require.Contains(t, err.Error(), "parsing registry hostname")
			require.NotContains(t, err.Error(), secret)
		})
	}
}

// mockResource implements regauthn.Resource for testing
type mockResource struct{}

func (m *mockResource) String() string {
	return "gcr.io/test/image"
}

func (m *mockResource) RegistryStr() string {
	return "gcr.io"
}
