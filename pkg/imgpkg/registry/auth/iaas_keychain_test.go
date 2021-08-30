package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeatureFlagValues(t *testing.T) {
	t.Run("Given a non-bool value should error", func(t *testing.T) {
		_, err := NewIaasKeychain(func() []string {
			return []string{"IMGPKG_ENABLE_IAAS_AUTH=non-bool-value"}
		})

		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "Expected a bool value (true, false). Got non-bool-value")
		}
	})
}
