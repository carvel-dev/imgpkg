// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagelayers

import (
	"github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"
	"testing"
)

func TestIncludesNonDistributableLayerWhenFlagIsProvided(t *testing.T) {
	imageLayer := imagedesc.ImageLayerDescriptor{
		MediaType: "application/vnd.oci.image.layer.nondistributable.v1.tar",
	}

	distributableFlag := true
	shouldWrite, err := ImageLayerWriterChecker{distributableFlag}.ShouldLayerBeIncluded(imagedesc.NewDescribedLayer(imageLayer, nil))
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
	shouldWrite, err := ImageLayerWriterChecker{distributableFlag}.ShouldLayerBeIncluded(imagedesc.NewDescribedLayer(imageLayer, nil))
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
	shouldWrite, err := ImageLayerWriterChecker{distributableFlag}.ShouldLayerBeIncluded(imagedesc.NewDescribedLayer(imageLayer, nil))
	if err != nil {
		t.Fatalf("Expected checking layer to succeed but got an error: %s", err)
	}

	if shouldWrite != true {
		t.Fatalf("Expected to return true, but instead returned false")
	}

	distributableFlag = true
	shouldWrite, err = ImageLayerWriterChecker{distributableFlag}.ShouldLayerBeIncluded(imagedesc.NewDescribedLayer(imageLayer, nil))
	if err != nil {
		t.Fatalf("Expected checking layer to succeed but got an error: %s", err)
	}

	if shouldWrite != true {
		t.Fatalf("Expected to return true, but instead returned false")
	}
}
