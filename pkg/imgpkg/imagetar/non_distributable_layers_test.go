// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/imagedesc"
)

func TestIncludesNonDistributableLayerWhenFlagIsProvided(t *testing.T) {
	imageLayer := imagedesc.ImageLayerDescriptor{
		MediaType: "application/vnd.oci.image.layer.nondistributable.v1.tar",
	}

	distributableFlag := true
	shouldWrite, err := ImageLayerWriterFilter{distributableFlag}.ShouldLayerBeIncluded(imagedesc.NewDescribedCompressedLayer(imageLayer, nil))
	if err != nil {
		t.Fatalf("Expected checking layer to succeed but got an error: %s", err)
	}

	if shouldWrite != true {
		t.Fatalf("Expected to return true, but instead returned false")
	}
}

func TestDoesNotIncludeNonDistributableLayerWhenFlagIsNotProvided(t *testing.T) {
	imageLayer := imagedesc.ImageLayerDescriptor{
		MediaType: "application/vnd.oci.image.layer.nondistributable.v1.tar",
	}

	distributableFlag := false
	shouldWrite, err := ImageLayerWriterFilter{distributableFlag}.ShouldLayerBeIncluded(imagedesc.NewDescribedCompressedLayer(imageLayer, nil))
	if err != nil {
		t.Fatalf("Expected checking layer to succeed but got an error: %s", err)
	}

	if shouldWrite != false {
		t.Fatalf("Expected to return false, but instead returned true")
	}
}

func TestIncludesDistributableLayer(t *testing.T) {
	imageLayer := imagedesc.ImageLayerDescriptor{}

	distributableFlag := false
	shouldWrite, err := ImageLayerWriterFilter{distributableFlag}.ShouldLayerBeIncluded(imagedesc.NewDescribedCompressedLayer(imageLayer, nil))
	if err != nil {
		t.Fatalf("Expected checking layer to succeed but got an error: %s", err)
	}

	if shouldWrite != true {
		t.Fatalf("Expected to return true, but instead returned false")
	}

	distributableFlag = true
	shouldWrite, err = ImageLayerWriterFilter{distributableFlag}.ShouldLayerBeIncluded(imagedesc.NewDescribedCompressedLayer(imageLayer, nil))
	if err != nil {
		t.Fatalf("Expected checking layer to succeed but got an error: %s", err)
	}

	if shouldWrite != true {
		t.Fatalf("Expected to return true, but instead returned false")
	}
}
