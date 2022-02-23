// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"sort"
	"strings"

	ctlbundle "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/signature"
)

// ImageType defines the type of Image
type ImageType string

const (
	// ContentImage Image that is part of the Bundle
	ContentImage ImageType = "Image"
	// SignatureImage Image that contains a signature
	SignatureImage ImageType = "Signature"
)

// DescribeOpts Options used when calling the Describe function
type DescribeOpts struct {
	Logger      Logger
	Concurrency int
}

// Author information from a Bundle
type Author struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// Website URL where more information of the Bundle can be found
type Website struct {
	URL string `json:"url,omitempty"`
}

// BundleMetadata Extra metadata present in a Bundle
type BundleMetadata struct {
	Metadata map[string]string `json:"metadata,omitempty"`
	Authors  []Author          `json:"authors,omitempty"`
	Websites []Website         `json:"websites,omitempty"`
}

// ImageInfo URLs where the image can be found as well as annotations provided in the Images Lock
type ImageInfo struct {
	Image       string            `json:"image"`
	Origin      string            `json:"origin"`
	Annotations map[string]string `json:"annotations,omitempty"`
	ImageType   ImageType         `json:"imageType"`
}

// BundleContent Contents present in a Bundle
type BundleContent struct {
	Bundles []BundleDescription `json:"bundles,omitempty"`
	Images  []ImageInfo         `json:"images,omitempty"`
}

// BundleDescription Metadata and Contents of a Bundle
type BundleDescription struct {
	Image       string            `json:"image"`
	Origin      string            `json:"origin"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Metadata    BundleMetadata    `json:"metadata,omitempty"`
	Content     BundleContent     `json:"content"`
}

// DescribeBundle Given a Bundle URL fetch the information about the contents of the Bundle and Nested Bundles
func DescribeBundle(bundleImage string, opts DescribeOpts, registryOpts registry.Opts) (BundleDescription, error) {
	reg, err := registry.NewSimpleRegistry(registryOpts)
	if err != nil {
		return BundleDescription{}, err
	}
	bundle := ctlbundle.NewBundle(bundleImage, reg)

	isBundle, err := bundle.IsBundle()
	if err != nil {
		return BundleDescription{}, err
	}

	if !isBundle {
		return BundleDescription{}, fmt.Errorf("The image %s is not a bundle", bundleImage)
	}

	signatureRetriever := signature.NewSignatures(signature.NewCosign(reg), opts.Concurrency)

	allBundles, allImgRefs, err := bundle.AllImagesRefs(opts.Concurrency, opts.Logger)
	if err != nil {
		return BundleDescription{}, err
	}

	bImageRefs := allImgRefs.ImageRefs()
	imgRefs := []lockconfig.ImageRef{{
		Image: bundle.DigestRef(),
	}}
	for _, ref := range bImageRefs {
		imgRefs = append(imgRefs, ref.ImageRef)
	}

	signatures, err := signatureRetriever.FetchFromImageRef(imgRefs)
	if err != nil {
		return BundleDescription{}, err
	}

	topBundle := refWithDescription{
		imgRef: ctlbundle.ImageRef{
			ImageRef: lockconfig.ImageRef{
				Image: bundle.DigestRef(),
			},
			IsBundle: &isBundle,
		},
	}
	return topBundle.DescribeBundle(allBundles, signatures), nil
}

type refWithDescription struct {
	imgRef ctlbundle.ImageRef
	bundle BundleDescription
}

func (r *refWithDescription) DescribeBundle(bundles []*ctlbundle.Bundle, signatures map[string]lockconfig.ImageRef) BundleDescription {
	var visitedImgs map[string]refWithDescription
	return r.describeBundleRec(visitedImgs, r.imgRef, bundles, signatures)
}

func (r *refWithDescription) describeBundleRec(visitedImgs map[string]refWithDescription, currentBundle ctlbundle.ImageRef, bundles []*ctlbundle.Bundle, signatures map[string]lockconfig.ImageRef) BundleDescription {
	desc, wasVisited := visitedImgs[currentBundle.Image]
	if wasVisited {
		return desc.bundle
	}

	desc = refWithDescription{
		imgRef: currentBundle,
		bundle: BundleDescription{
			Image:       currentBundle.PrimaryLocation(),
			Origin:      currentBundle.Image,
			Annotations: currentBundle.Annotations,
			Metadata:    BundleMetadata{},
			Content:     BundleContent{},
		},
	}
	var bundle *ctlbundle.Bundle
	for _, b := range bundles {
		if b.DigestRef() == currentBundle.PrimaryLocation() {
			bundle = b
			break
		}
	}
	if bundle == nil {
		panic("Internal consistency: bundle could not be found in list of bundles")
	}

	imagesRefs := bundle.ImagesRefs()
	sort.Slice(imagesRefs, func(i, j int) bool {
		return strings.Compare(imagesRefs[i].Image, imagesRefs[j].Image) <= 0
	})

	for _, ref := range imagesRefs {
		if ref.IsBundle == nil {
			panic("Internal consistency: IsBundle after processing must always have a value")
		}

		if *ref.IsBundle {
			bundleDesc := r.describeBundleRec(visitedImgs, ref, bundles, signatures)
			desc.bundle.Content.Bundles = append(desc.bundle.Content.Bundles, bundleDesc)
		} else {
			desc.bundle.Content.Images = append(desc.bundle.Content.Images, ImageInfo{
				Image:       ref.PrimaryLocation(),
				Origin:      ref.Image,
				Annotations: ref.Annotations,
				ImageType:   ContentImage,
			})
		}

		if sig, ok := signatures[ref.PrimaryLocation()]; ok {
			desc.bundle.Content.Images = append(desc.bundle.Content.Images, ImageInfo{
				Image:       sig.PrimaryLocation(),
				Annotations: sig.Annotations,
				ImageType:   SignatureImage,
			})
		}
	}

	if sig, ok := signatures[currentBundle.PrimaryLocation()]; ok {
		desc.bundle.Content.Images = append(desc.bundle.Content.Images, ImageInfo{
			Image:       sig.PrimaryLocation(),
			Annotations: sig.Annotations,
			ImageType:   SignatureImage,
		})
	}

	return desc.bundle
}
