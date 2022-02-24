// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"sort"
	"strings"

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

// Author information from a Bundle
type Author struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// Website URL where more information of the Bundle can be found
type Website struct {
	URL string `json:"url,omitempty"`
}

// Metadata Extra metadata present in a Bundle
type Metadata struct {
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

// Content Contents present in a Bundle
type Content struct {
	Bundles []Description `json:"bundles,omitempty"`
	Images  []ImageInfo   `json:"images,omitempty"`
}

// Description Metadata and Contents of a Bundle
type Description struct {
	Image       string            `json:"image"`
	Origin      string            `json:"origin"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Metadata    Metadata          `json:"metadata,omitempty"`
	Content     Content           `json:"content"`
}

// DescribeOpts Options used when calling the Describe function
type DescribeOpts struct {
	Logger      Logger
	Concurrency int
}

type SignatureFetcher interface {
	FetchFromImageRef(images []lockconfig.ImageRef) (map[string]lockconfig.ImageRef, error)
}

// Describe Given a Bundle URL fetch the information about the contents of the Bundle and Nested Bundles
func Describe(bundleImage string, opts DescribeOpts, registryOpts registry.Opts) (Description, error) {
	reg, err := registry.NewSimpleRegistry(registryOpts)
	if err != nil {
		return Description{}, err
	}

	signatureRetriever := signature.NewSignatures(signature.NewCosign(reg), opts.Concurrency)

	return DescribeWithRegistryAndSignatureFetcher(bundleImage, opts, reg, signatureRetriever)
}

// DescribeWithRegistryAndSignatureFetcher Given a Bundle URL fetch the information about the contents of the Bundle and Nested Bundles
func DescribeWithRegistryAndSignatureFetcher(bundleImage string, opts DescribeOpts, reg ImagesMetadata, sigFetcher SignatureFetcher) (Description, error) {
	bundle := NewBundle(bundleImage, reg)
	allBundles, allImgRefs, err := bundle.AllImagesRefs(opts.Concurrency, opts.Logger)
	if err != nil {
		return Description{}, err
	}

	bImageRefs := allImgRefs.ImageRefs()
	imgRefs := []lockconfig.ImageRef{{
		Image: bundle.DigestRef(),
	}}
	for _, ref := range bImageRefs {
		imgRefs = append(imgRefs, ref.ImageRef)
	}

	signatures, err := sigFetcher.FetchFromImageRef(imgRefs)
	if err != nil {
		return Description{}, err
	}

	isBundle := true
	topBundle := refWithDescription{
		imgRef: ImageRef{
			ImageRef: lockconfig.ImageRef{
				Image: bundle.DigestRef(),
			},
			IsBundle: &isBundle,
		},
	}
	return topBundle.DescribeBundle(allBundles, signatures), nil
}

type refWithDescription struct {
	imgRef ImageRef
	bundle Description
}

func (r *refWithDescription) DescribeBundle(bundles []*Bundle, signatures map[string]lockconfig.ImageRef) Description {
	var visitedImgs map[string]refWithDescription
	return r.describeBundleRec(visitedImgs, r.imgRef, bundles, signatures)
}

func (r *refWithDescription) describeBundleRec(visitedImgs map[string]refWithDescription, currentBundle ImageRef, bundles []*Bundle, signatures map[string]lockconfig.ImageRef) Description {
	desc, wasVisited := visitedImgs[currentBundle.Image]
	if wasVisited {
		return desc.bundle
	}

	desc = refWithDescription{
		imgRef: currentBundle,
		bundle: Description{
			Image:       currentBundle.PrimaryLocation(),
			Origin:      currentBundle.Image,
			Annotations: currentBundle.Annotations,
			Metadata:    Metadata{},
			Content:     Content{},
		},
	}
	var bundle *Bundle
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
