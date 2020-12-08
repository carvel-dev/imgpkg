// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package lockfiles

import (
	"testing"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	fake "github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
)

func TestIsBundle_ImageIsBundle(t *testing.T) {
	labels := make(map[string]string)
	labels[image.BundleConfigLabel] = "foo"

	img := fakeImage(labels)

	isBundle, err := IsBundle(img)
	if err != nil {
		t.Fatalf("Expected no error but received one: %v", err)
	}

	if !isBundle {
		t.Fatalf("Expected image to be valid Bundle but was not")
	}
}

func TestIsBundle_ImageIsNotBundle(t *testing.T) {
	labels := make(map[string]string)
	img := fakeImage(labels)

	isBundle, err := IsBundle(img)
	if err != nil {
		t.Fatalf("Expected no error but received one: %v", err)
	}

	if isBundle {
		t.Fatalf("Expected image to be invalid Bundle but was not")
	}
}

func fakeImage(labels map[string]string) regv1.Image {
	img := &fake.FakeImage{}
	img.ConfigFileReturns(&regv1.ConfigFile{
		Config: regv1.Config{
			Labels: labels,
		},
	}, nil)

	return img
}
