// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package imagedesc_test

import (
	"strings"
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/imagedesc"
)

func TestNewImageRefDescriptorsFromBytes(t *testing.T) {
	t.Run("rejects an image descriptor with no refs", func(t *testing.T) {
		manifest := `[{"Image":{"Refs":[],"Manifest":{"Digest":"sha256:aaaa"}}}]`

		_, err := imagedesc.NewImageRefDescriptorsFromBytes([]byte(manifest))
		if err == nil {
			t.Fatal("expected an error, got none")
		}
		if !strings.Contains(err.Error(), "at least one ref") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("rejects an image nested in an index with no refs", func(t *testing.T) {
		manifest := `[{"ImageIndex":{"Refs":["registry.example.com/repo@sha256:bbbb"],"Digest":"sha256:bbbb","Images":[{"Refs":[],"Manifest":{"Digest":"sha256:aaaa"}}]}}]`

		_, err := imagedesc.NewImageRefDescriptorsFromBytes([]byte(manifest))
		if err == nil {
			t.Fatal("expected an error, got none")
		}
		if !strings.Contains(err.Error(), "at least one ref") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("accepts a descriptor with a ref", func(t *testing.T) {
		manifest := `[{"Image":{"Refs":["registry.example.com/repo@sha256:aaaa"],"Manifest":{"Digest":"sha256:aaaa"}}}]`

		ids, err := imagedesc.NewImageRefDescriptorsFromBytes([]byte(manifest))
		if err != nil {
			t.Fatalf("got an error: %v", err)
		}
		if got := len(ids.Descriptors()); got != 1 {
			t.Errorf("expected 1 descriptor, got %d", got)
		}
	})
}
