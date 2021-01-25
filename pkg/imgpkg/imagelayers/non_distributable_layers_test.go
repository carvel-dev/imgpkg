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
	shouldWrite := ImageLayerWriterChecker{distributableFlag}.ShouldLayerBeIncluded(imageLayer)

	if shouldWrite != true {
		t.Fatalf("Expected to return true, but instead returned false")
	}
}

func TestDoesNotIncludeNonDistributableLayerWhenFlagIsNotProvided(t *testing.T) {
	imageLayer := imagedesc.ImageLayerDescriptor{
		MediaType: "application/vnd.oci.image.layer.nondistributable.v1.tar",
	}

	distributableFlag := false
	shouldWrite := ImageLayerWriterChecker{distributableFlag}.ShouldLayerBeIncluded(imageLayer)

	if shouldWrite != false {
		t.Fatalf("Expected to return false, but instead returned true")
	}
}

func TestIncludesDistributableLayer(t *testing.T) {
	imageLayer := imagedesc.ImageLayerDescriptor{}

	distributableFlag := false
	shouldWrite := ImageLayerWriterChecker{distributableFlag}.ShouldLayerBeIncluded(imageLayer)

	if shouldWrite != true {
		t.Fatalf("Expected to return true, but instead returned false")
	}

	distributableFlag = true
	shouldWrite = ImageLayerWriterChecker{distributableFlag}.ShouldLayerBeIncluded(imageLayer)

	if shouldWrite != true {
		t.Fatalf("Expected to return true, but instead returned false")
	}
}
