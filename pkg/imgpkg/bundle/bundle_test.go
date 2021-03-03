package bundle_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/stretchr/testify/assert"
)

func TestPullWritingContentsToDisk(t *testing.T) {
	fakeUI := &bundlefakes.FakeUI{}

	t.Run("bundle referencing an image", func(t *testing.T) {
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
		expectedConfigFile, err := os.ReadFile("test_assets/bundle/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))
	})

	t.Run("bundle referencing another bundle", func(t *testing.T) {
		fakeRegistry := NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle
		fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle("repo/bundle_icecream_with_single_bundle", fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, fakeUI)
		assert.NoError(t, err)

		assert.DirExists(t, outputPath)
		digest, err := fakeRegistry.state["index.docker.io/icecream/bundle:latest"].image.Digest()
		assert.NoError(t, err)

		outputDirConfigFile := filepath.Join(outputPath, ".imgpkg", "bundles", digest.String(), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err := os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err := os.ReadFile("test_assets/bundle_with_mult_images/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))
	})

	t.Run("bundle referencing another bundle that references another bundle", func(t *testing.T) {
		// setup
		fakeRegistry := NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle - dependsOn - apples/bundle
		fakeRegistry.WithBundleFromPath("apples/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_apples_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_apples_with_single_bundle", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle("repo/bundle_icecream_with_single_bundle", fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		// test subject
		err = subject.Pull(outputPath, fakeUI)
		assert.NoError(t, err)

		// assert icecream bundle was recursively pulled onto disk
		assert.DirExists(t, outputPath)
		digest, err := fakeRegistry.state["index.docker.io/icecream/bundle:latest"].image.Digest()
		assert.NoError(t, err)

		outputDirConfigFile := filepath.Join(outputPath, ".imgpkg", "bundles", digest.String(), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err := os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err := os.ReadFile("test_assets/bundle_apples_with_single_bundle/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))

		// assert apples bundle was recursively pulled onto disk
		digest, err = fakeRegistry.state["index.docker.io/apples/bundle:latest"].image.Digest()
		assert.NoError(t, err)

		outputDirConfigFile = filepath.Join(outputPath, ".imgpkg", "bundles", digest.String(), "config.yml")
		assert.FileExists(t, outputDirConfigFile)
		actualConfigFile, err = os.ReadFile(outputDirConfigFile)
		assert.NoError(t, err)
		expectedConfigFile, err = os.ReadFile("test_assets/bundle_with_mult_images/config.yml")
		assert.NoError(t, err)
		assert.Equal(t, string(actualConfigFile), string(expectedConfigFile))
	})
}

func TestPullOutput(t *testing.T) {
	t.Run("bundle referencing another bundle", func(t *testing.T) {
		output := bytes.NewBufferString("")
		writerUI := ui.NewWriterUI(output, output, nil)
		fakeRegistry := NewFakeRegistry(t)
		defer fakeRegistry.CleanUp()

		// repo/bundle_icecream_with_single_bundle - dependsOn - icecream/bundle
		fakeRegistry.WithBundleFromPath("icecream/bundle", "test_assets/bundle_with_mult_images").WithEveryImageFrom("test_assets/image_with_config", map[string]string{})
		fakeRegistry.WithBundleFromPath("repo/bundle_icecream_with_single_bundle", "test_assets/bundle_icecream_with_single_bundle").WithEveryImageFrom("test_assets/bundle_with_mult_images", map[string]string{"dev.carvel.imgpkg.bundle": ""})

		subject := bundle.NewBundle("repo/bundle_icecream_with_single_bundle", fakeRegistry.Build())
		outputPath, err := os.MkdirTemp(os.TempDir(), "test-output-bundle-path")
		assert.NoError(t, err)
		defer os.Remove(outputPath)

		err = subject.Pull(outputPath, writerUI)
		assert.NoError(t, err)

		assert.Regexp(t,
`Pulling bundle 'index.docker.io/repo/bundle_icecream_with_single_bundle@sha256:.*'
Extracting layer 'sha256:.*' \(1/1\)
Nested bundles
  Pulling Nested bundle 'index.docker.io/icecream/bundle@sha256:.*'
  Extracting layer 'sha256:.*' \(1/1\)
Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update
Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update`, output.String())
	})
}

/*
Pulling bundle 'my.registry.io/r-bundle@sha256:ccccccccccfdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0'
Bundle Layers
  Extracting layer 'sha256:87bf2c587b3315143cd05df7bd24d4e608ddb59f8c62110fe1b579fb817a2917' (1/1)

Nested bundles
  Pulling Bundle 'my.registry.io/bundle-1@sha256:aaaaaaaaaafdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0' (1/2)
  Extracting layer 'sha256:81fc6f37c9774541136e6113d899c215151496f4cf91c89c056783d2feb5ae0d' (1/1)
    Found 1 Bundle packaged

    Pulling Nested Bundle 'my.registry.io/bundle-2@sha256:ddddddddddfdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0' (1/1)
    Extracting layer 'sha256:9abb11371e7e53b5c33da086ea50dabb5d4cdd280be7d489169374b0188feab1' (1/1)

  Pulling Nested Bundle 'my.registry.io/bundle-2@sha256:ddddddddddfdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0' (2/2)
  Skipped, already downloaded

Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update
*/
