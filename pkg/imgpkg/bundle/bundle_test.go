package bundle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/stretchr/testify/assert"
)

func TestPull(t *testing.T) {
	fakeUI := &bundlefakes.FakeUI{}

	t.Run("a single bundle referencing an image", func(t *testing.T) {
		fakeRegistry := NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		fakeRegistry.WithBundleFromPath("repo/some-bundle-name", "test_assets/bundle").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		subject := bundle.NewBundle("repo/some-bundle-name", fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		outputDirConfigFile := filepath.Join(outputPath, "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err := os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err := os.ReadFile("test_assets/image_with_config/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, actualConfigFile, expectedConfigFile)
	})

	t.Run("a single bundle referencing another bundle", func(t *testing.T) {
		fakeRegistry := NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		fakeRegistry.WithBundleFromPath("repo/bundle-with-a-bundle", "test_assets/bundle_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeRegistry.WithBundleFromPath("repo/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})

		subject := bundle.NewBundle("repo/bundle-with-a-bundle", fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		digest, err := fakeRegistry.state["index.docker.io/repo/bundle:latest"].image.Digest()
		assert.NoError(t, err)
		outputDirConfigFile := filepath.Join(outputPath, ".imgpkg", "bundles", digest.String(), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err := os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err := os.ReadFile("test_assets/image_with_config/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, actualConfigFile, expectedConfigFile)
	})
}
