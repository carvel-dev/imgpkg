// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package lockconfig_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
)

func TestNewImagesLockFromBytes(t *testing.T) {
	t.Run("When image reference is not resolved, it errors", func(t *testing.T) {
		data := `
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
- image: nginx:v1
`

		_, err := lockconfig.NewImagesLockFromBytes([]byte(data))
		require.EqualError(t, err, "Validating images lock: Expected ref to be in digest form, got 'nginx:v1'")
	})

	t.Run("when yaml contain keys that are unknown, it errors", func(t *testing.T) {
		data := `
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
another-key: invalid
`

		_, err := lockconfig.NewImagesLockFromBytes([]byte(data))
		require.Error(t, err)
		require.Contains(t, err.Error(), `unknown field "another-key"`)
	})
}

func TestAddImageRef(t *testing.T) {
	data := `
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
`

	subject, err := lockconfig.NewImagesLockFromBytes([]byte(data))
	require.NoError(t, err)

	t.Run("when locations are present, maintain locations", func(t *testing.T) {
		subject := subject // copy
		imgRef := lockconfig.ImageRef{
			Image: "some.image.io/test@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0",
		}
		imgRef.AddLocation("other.registry.io/test@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0")
		imgRef.AddLocation("some-other.registry.io/othername@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0")
		subject.AddImageRef(imgRef)
		require.Len(t, subject.Images, 1)

		assert.Len(t, subject.Images[0].Locations(), 3)
		assert.Contains(t, subject.Images[0].Locations(), "other.registry.io/test@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0")
		assert.Contains(t, subject.Images[0].Locations(), "some-other.registry.io/othername@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0")
		assert.Contains(t, subject.Images[0].Locations(), "some.image.io/test@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0")
	})

	t.Run("when image ref as no location, it returns only the Image current known location", func(t *testing.T) {
		subject := subject // copy
		imgRef := lockconfig.ImageRef{
			Image: "some.image.io/test@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0",
		}
		subject.AddImageRef(imgRef)
		require.Len(t, subject.Images, 1)

		assert.Len(t, subject.Images[0].Locations(), 1)
		assert.Contains(t, subject.Images[0].Locations(), "some.image.io/test@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0")
	})
}
