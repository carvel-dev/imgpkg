// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	credentialprovider "github.com/vdemeester/k8s-pkg-credentialprovider"
)

func TestFeatureFlagValues(t *testing.T) {
	t.Run("Given a non-bool value should error", func(t *testing.T) {
		_, err := NewIaasKeychain(context.Background(), func() []string {
			return []string{"IMGPKG_ENABLE_IAAS_AUTH=non-bool-value"}
		})

		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Expected IMGPKG_ENABLE_IAAS_AUTH to contain a boolean value (true, false). Got non-bool-value")
		}
	})
}

func TestTimeoutWhenEnablingProvider(t *testing.T) {
	t.Run("Should timeout if gcp metadata service is not responsive. See https://github.com/tektoncd/pipeline/issues/1742#issuecomment-565055556", func(t *testing.T) {
		blockingDockerProvider := registerBlockingProvider()

		defer func() {
			close(blockingDockerProvider.shouldStopBlocking)
		}()

		require.Eventually(t, func() bool {
			timeoutQuickly, cancelImmediately := context.WithTimeout(context.Background(), 1*time.Second)
			cancelImmediately()

			_, err := NewIaasKeychain(timeoutQuickly, func() []string {
				return []string{}
			})

			if assert.Error(t, err) {
				assert.Equal(t, "Timeout occurred trying to enable IaaS provider. (hint: To skip authenticating via IaaS set the environment variable IMGPKG_ENABLE_IAAS_AUTH=false)", err.Error())
			}

			return true
		}, 5*time.Second, 1*time.Second)
	})
}

func registerBlockingProvider() *blockingProvider {
	blockingTestProvider := &blockingProvider{
		shouldStopBlocking: make(chan struct{}),
	}
	credentialprovider.RegisterCredentialProvider("TEST-blocking-dockercfg-TEST",
		&credentialprovider.CachingDockerConfigProvider{
			Provider: blockingTestProvider,
		})

	return blockingTestProvider
}

type blockingProvider struct {
	shouldStopBlocking chan struct{}
}

func (a *blockingProvider) Enabled() bool {
	select {
	case <-a.shouldStopBlocking:
		return true
	}
}

func (a blockingProvider) Provide(_ string) credentialprovider.DockerConfig {
	return credentialprovider.DockerConfig{}
}
