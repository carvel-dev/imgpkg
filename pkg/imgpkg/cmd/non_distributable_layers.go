// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagedesc"
	ctlimgset "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
)

type nonDistributableLayers struct {
	ImgRef     string
	ImgFullRef string
	Layers     []string
}

func getNonDistributableLayersFromImageDescriptors(ids *imagedesc.ImageRefDescriptors) []nonDistributableLayers {
	var nonDistLayers []nonDistributableLayers

	for _, descriptor := range ids.Descriptors() {
		if descriptor.Image != nil {
			nonDistLayers = append(nonDistLayers, nonDistributableLayersFromImageDescriptors(*descriptor.Image)...)
		} else if descriptor.ImageIndex != nil {
			nonDistLayers = append(nonDistLayers, nonDistributableLayersFromIndexDescriptors(*descriptor.ImageIndex)...)
		}
	}
	return nonDistLayers
}

func nonDistributableLayersFromImageDescriptors(descriptor imagedesc.ImageDescriptor) []nonDistributableLayers {
	var nonDistLayers []nonDistributableLayers
	imgLayers := nonDistributableLayers{
		ImgFullRef: descriptor.Refs[0],
	}
	for _, layerDescriptor := range descriptor.Layers {
		if !layerDescriptor.IsDistributable() {
			imgLayers.Layers = append(imgLayers.Layers, layerDescriptor.Digest)
		}
	}
	if len(imgLayers.Layers) > 0 {
		nonDistLayers = append(nonDistLayers, imgLayers)
	}
	return nonDistLayers
}

func nonDistributableLayersFromIndexDescriptors(descriptor imagedesc.ImageIndexDescriptor) []nonDistributableLayers {
	var nonDistLayers []nonDistributableLayers
	for _, image := range descriptor.Images {
		nonDistLayers = append(nonDistLayers, nonDistributableLayersFromImageDescriptors(image)...)
	}
	for _, index := range descriptor.Indexes {
		nonDistLayers = append(nonDistLayers, nonDistributableLayersFromIndexDescriptors(index)...)
	}
	return nonDistLayers
}

func processedImagesNonDistLayer(processedImages *ctlimgset.ProcessedImages) []nonDistributableLayers {
	var everyNonDistImages []nonDistributableLayers
	for _, image := range processedImages.All() {
		ref, err := name.ParseReference(image.DigestRef)
		if err != nil {
			panic(fmt.Sprintf("Internal consistency: '%s' should be a valid reference: %s", image.DigestRef, err))
		}
		if image.ImageIndex != nil {
			imagesNonDistributableLayers := everyNonDistributableLayerForAnImageIndex(image.ImageIndex)
			for i, layer := range imagesNonDistributableLayers {
				digestRef := ref.Context().Digest(layer.ImgRef)
				imagesNonDistributableLayers[i].ImgFullRef = digestRef.String()
			}
			everyNonDistImages = append(everyNonDistImages, imagesNonDistributableLayers...)
		} else if image.Image != nil {
			imagesNonDistributableLayers := everyNonDistributableLayerForAnImage(image.Image)
			for i, layer := range imagesNonDistributableLayers {
				digestRef := ref.Context().Digest(layer.ImgRef)
				imagesNonDistributableLayers[i].ImgFullRef = digestRef.String()
			}
			everyNonDistImages = append(everyNonDistImages, imagesNonDistributableLayers...)
		}
	}

	return everyNonDistImages
}

func everyNonDistributableLayerForAnImageIndex(imageIndex regv1.ImageIndex) []nonDistributableLayers {
	var everyNonDistImages []nonDistributableLayers
	indexManifest, err := imageIndex.IndexManifest()
	if err != nil {
		return nil
	}
	for _, descriptor := range indexManifest.Manifests {
		if descriptor.MediaType.IsIndex() {
			imageIndex, err := imageIndex.ImageIndex(descriptor.Digest)
			if err != nil {
				continue
			}
			mediaTypesForImageIndex := everyNonDistributableLayerForAnImageIndex(imageIndex)
			everyNonDistImages = append(everyNonDistImages, mediaTypesForImageIndex...)
		} else {
			image, err := imageIndex.Image(descriptor.Digest)
			if err != nil {
				continue
			}
			mediaTypeForImage := everyNonDistributableLayerForAnImage(image)
			everyNonDistImages = append(everyNonDistImages, mediaTypeForImage...)
		}
	}

	return everyNonDistImages
}

func everyNonDistributableLayerForAnImage(image regv1.Image) []nonDistributableLayers {
	digest, err := image.Digest()
	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency: cannot retrieve digest from image"))
	}
	imgLayers := nonDistributableLayers{
		ImgRef: digest.String(),
	}
	layers, err := image.Layers()
	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency: cannot retrieve layers from image '%s'", digest))
	}

	for _, layerDescriptor := range layers {
		mediaType, err := layerDescriptor.MediaType()
		if err != nil {
			continue
		}
		if !mediaType.IsDistributable() {
			hash, err := layerDescriptor.Digest()
			if err != nil {
				continue
			}
			imgLayers.Layers = append(imgLayers.Layers, hash.String())
		}
	}
	if len(imgLayers.Layers) > 0 {
		return []nonDistributableLayers{imgLayers}
	}
	return nil
}

func informUserToUseTheNonDistributableFlagWithDescriptors(ui util.LoggerWithLevels, includeNonDistributableFlag bool, everyImageWithNonDistLayer []nonDistributableLayers) {
	if includeNonDistributableFlag && len(everyImageWithNonDistLayer) == 0 {
		ui.Warnf("'--include-non-distributable-layers' flag provided, but no images contained a non-distributable layer.\n")
	} else if !includeNonDistributableFlag && len(everyImageWithNonDistLayer) > 0 {
		msg := "Skipped the followings layer(s) due to it being non-distributable. If you would like to include non-distributable layers, use the --include-non-distributable-layers flag"
		for _, img := range everyImageWithNonDistLayer {
			msg += "\n - Image: " + img.ImgFullRef
			msg += "\n   Layers:"
			for _, layer := range img.Layers {
				msg += "\n     - " + layer
			}
		}
		ui.Warnf(msg)
	}
}
