// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"testing"

	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/stretchr/testify/require"
)

func TestNewLocationConfigFromBytes(t *testing.T) {
	t.Run("When API version is different, it fails", func(t *testing.T) {
		data := `
apiVersion: imgpkg.carvel.dev/v1alpha2
kind: ImageLocations
images:
- image: some.image.io/test@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
  isBundle: true
`

		_, err := bundle.NewLocationConfigFromBytes([]byte(data))
		require.EqualError(t, err, "Validating apiVersion: Unknown version (known: imgpkg.carvel.dev/v1alpha1)")
	})

	t.Run("When unknown fields are present, it returns the locations configuration", func(t *testing.T) {
		data := `
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImageLocations
images:
- image: some.image.io/test@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
  isBundle: true
  some-other-key: value
`

		_, err := bundle.NewLocationConfigFromBytes([]byte(data))
		require.NoError(t, err)
	})
}
