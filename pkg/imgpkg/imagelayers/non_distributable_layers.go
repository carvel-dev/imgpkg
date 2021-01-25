// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagelayers

import "github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"

type ImageLayerWriterChecker struct {
	includeNonDistributable bool
}

func NewImageLayerWriterCheck(includeNonDistributable bool) ImageLayerWriterChecker {
	return ImageLayerWriterChecker{includeNonDistributable}
}

func (c ImageLayerWriterChecker) ShouldLayerBeIncluded(layer imagedesc.ImageLayerDescriptor) bool {
	return layer.IsDistributable() || c.includeNonDistributable
}
