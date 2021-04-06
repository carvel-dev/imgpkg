package image

import (
	"os"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
)

func TestAnonAuthWhenNoEnvVarsProvided(t *testing.T) {
	envKeychain := NewEnvKeychain("IMGPKG_")
	resource, err := name.NewRepository("imgpkg_test")
	assert.NoError(t, err)

	auth, err := envKeychain.Resolve(resource)
	assert.NoError(t, err)

	assert.Equal(t, authn.Anonymous, auth)
}

func TestEnvAuthWhenEnvVarsProvided(t *testing.T) {
	user := "my_cool_user"
	pass := "my_neat_pass"
	err := os.Setenv("TEST_IMGPKG_REGISTRY_USERNAME", user)
	assert.NoError(t, err)
	err = os.Setenv("TEST_IMGPKG_REGISTRY_PASSWORD", pass)
	assert.NoError(t, err)
	err = os.Setenv("TEST_IMGPKG_REGISTRY_HOSTNAME", "localhost:9999")
	assert.NoError(t, err)

	defer os.Unsetenv("TEST_IMGPKG_REGISTRY_USERNAME")
	defer os.Unsetenv("TEST_IMGPKG_REGISTRY_PASSWORD")
	defer os.Unsetenv("TEST_IMGPKG_REGISTRY_HOSTNAME")

	envKeychain := NewEnvKeychain("TEST_IMGPKG_REGISTRY")
	resource, err := name.NewRepository("localhost:9999/imgpkg_test")
	assert.NoError(t, err)

	auth, err := envKeychain.Resolve(resource)
	assert.NoError(t, err)

	expected := authn.FromConfig(authn.AuthConfig{
		Username: user,
		Password: pass,
	})

	assert.Equal(t, expected, auth)
}

func TestEnvAuthWhenEnvVarsProvidedWithMultipleRegistries(t *testing.T) {
	err := os.Setenv("TEST_IMGPKG_REGISTRY_USERNAME_0", "user_0")
	assert.NoError(t, err)
	err = os.Setenv("TEST_IMGPKG_REGISTRY_PASSWORD_0", "pass_0")
	assert.NoError(t, err)
	err = os.Setenv("TEST_IMGPKG_REGISTRY_HOSTNAME_0", "localhost:0000")
	assert.NoError(t, err)

	err = os.Setenv("TEST_IMGPKG_REGISTRY_USERNAME_1", "user_1")
	assert.NoError(t, err)
	err = os.Setenv("TEST_IMGPKG_REGISTRY_PASSWORD_1", "pass_1")
	assert.NoError(t, err)
	err = os.Setenv("TEST_IMGPKG_REGISTRY_HOSTNAME_1", "localhost:1111")
	assert.NoError(t, err)


	defer func() {
		os.Unsetenv("TEST_IMGPKG_REGISTRY_USERNAME_0")
		os.Unsetenv("TEST_IMGPKG_REGISTRY_PASSWORD_0")
		os.Unsetenv("TEST_IMGPKG_REGISTRY_HOSTNAME_0")
		os.Unsetenv("TEST_IMGPKG_REGISTRY_USERNAME_1")
		os.Unsetenv("TEST_IMGPKG_REGISTRY_PASSWORD_1")
		os.Unsetenv("TEST_IMGPKG_REGISTRY_HOSTNAME_1")
	}()

	envKeychain := NewEnvKeychain("TEST_IMGPKG_REGISTRY")
	resource, err := name.NewRepository("localhost:1111/imgpkg_test")
	assert.NoError(t, err)

	auth, err := envKeychain.Resolve(resource)
	assert.NoError(t, err)

	expected := authn.FromConfig(authn.AuthConfig{
		Username: "user_1",
		Password: "pass_1",
	})

	assert.Equal(t, expected, auth)
}
