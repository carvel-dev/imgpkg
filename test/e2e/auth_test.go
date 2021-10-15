// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

func TestAuth(t *testing.T) {
	t.Run("Basic Auth", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}

		outputDir := env.Assets.CreateTempFolder("pull-image")
		defer env.Assets.CleanCreatedFolders()

		expectedUsername := "expected-user"
		expectedPassword := "expected-password"
		imageRef := "repo/imgpkg-test"

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		fakeRegistry.WithBasicAuth(expectedUsername, expectedPassword)
		fakeRegistry.WithRandomImage(imageRef)
		fakeRegistry.Build()

		_, err := imgpkg.RunWithOpts([]string{"pull", "-i", fakeRegistry.ReferenceOnTestServer(imageRef), "-o", outputDir}, helpers.RunOpts{
			EnvVars: []string{"IMGPKG_REGISTRY_HOSTNAME=" + fakeRegistry.Host(), "IMGPKG_REGISTRY_USERNAME=expected-user", "IMGPKG_REGISTRY_PASSWORD=expected-password"},
		})

		assert.NoError(t, err)
	})

	t.Run("Identity Token", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}

		outputDir := env.Assets.CreateTempFolder("pull-image")
		defer env.Assets.CleanCreatedFolders()

		expectedToken := "ID_TOKEN"
		imageRef := "repo/imgpkg-test"

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		fakeRegistry.WithIdentityToken(expectedToken)
		fakeRegistry.WithRandomImage(imageRef)
		fakeRegistry.Build()

		_, err := imgpkg.RunWithOpts([]string{"pull", "-i", fakeRegistry.ReferenceOnTestServer(imageRef), "-o", outputDir}, helpers.RunOpts{
			EnvVars: []string{"IMGPKG_REGISTRY_HOSTNAME=" + fakeRegistry.Host(), "IMGPKG_REGISTRY_IDENTITY_TOKEN=" + expectedToken},
		})

		assert.NoError(t, err)
	})

	t.Run("Identity Token", func(t *testing.T) {
		env := helpers.BuildEnv(t)
		imgpkg := helpers.Imgpkg{T: t, ImgpkgPath: env.ImgpkgPath}

		outputDir := env.Assets.CreateTempFolder("pull-image")
		defer env.Assets.CleanCreatedFolders()

		expectedToken := "REGISTRY_TOKEN"
		imageRef := "repo/imgpkg-test"

		fakeRegistry := helpers.NewFakeRegistry(t, &helpers.Logger{LogLevel: helpers.LogDebug})
		fakeRegistry.WithRegistryToken(expectedToken)
		fakeRegistry.WithRandomImage(imageRef)
		fakeRegistry.Build()

		_, err := imgpkg.RunWithOpts([]string{"pull", "-i", fakeRegistry.ReferenceOnTestServer(imageRef), "-o", outputDir}, helpers.RunOpts{
			EnvVars: []string{"IMGPKG_REGISTRY_HOSTNAME=" + fakeRegistry.Host(), "IMGPKG_REGISTRY_REGISTRY_TOKEN=" + expectedToken},
		})

		assert.NoError(t, err)
	})
}
