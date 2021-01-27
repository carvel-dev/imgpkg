// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagelayers

import (
	regv1 "github.com/google/go-containerregistry/pkg/v1"
)

type ImageLayerWriterChecker struct {
	includeNonDistributable bool
}

func NewImageLayerWriterCheck(includeNonDistributable bool) ImageLayerWriterChecker {
	return ImageLayerWriterChecker{includeNonDistributable}
}

func (c ImageLayerWriterChecker) ShouldLayerBeIncluded(layer regv1.Layer) (bool, error) {
	mediaType, err := layer.MediaType()
	if err != nil {
		return false, err
	}
	return mediaType.IsDistributable() || c.includeNonDistributable, nil
}
