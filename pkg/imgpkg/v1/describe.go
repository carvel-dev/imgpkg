// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package v1 contains the public API version 1 used by other tools to interact with imgpkg
package v1

import (
	"fmt"
	"sort"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/artifacts"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
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
	Image       string            `json:"image,omitempty"`
	Origin      string            `json:"origin,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	ImageType   bundle.ImageType  `json:"imageType"`
	Error       string            `json:"error,omitempty"`
}

// Content Contents present in a Bundle
type Content struct {
	Bundles map[string]Description `json:"bundles,omitempty"`
	Images  map[string]ImageInfo   `json:"images,omitempty"`
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
	Logger                 bundle.Logger
	Concurrency            int
	IncludeCosignArtifacts bool
}

// ArtifactFetcher Interface to retrieve signatures associated with Images
type ArtifactFetcher interface {
	FetchForImageRefs(images []lockconfig.ImageRef) ([]artifacts.ArtifactImageRef, error)
}

// Describe Given a Bundle URL fetch the information about the contents of the Bundle and Nested Bundles
func Describe(bundleImage string, opts DescribeOpts, registryOpts registry.Opts) (Description, error) {
	reg, err := registry.NewSimpleRegistry(registryOpts)
	if err != nil {
		return Description{}, err
	}

	var artifactFetcher ArtifactFetcher
	if !opts.IncludeCosignArtifacts {
		artifactFetcher = artifacts.NewNoop()
	} else {
		artifactFetcher = artifacts.NewArtifacts(artifacts.NewCosign(reg), opts.Concurrency)
	}

	return DescribeWithRegistryAndArtifactFetcher(bundleImage, opts, reg, artifactFetcher)
}

// DescribeWithRegistryAndArtifactFetcher Given a Bundle URL fetch the information about the contents of the Bundle and Nested Bundles
func DescribeWithRegistryAndArtifactFetcher(bundleImage string, opts DescribeOpts, reg bundle.ImagesMetadata, artifactFetcher ArtifactFetcher) (Description, error) {
	newBundle := bundle.NewBundle(bundleImage, reg)
	isBundle, err := newBundle.IsBundle()
	if err != nil {
		return Description{}, fmt.Errorf("Unable to check if %s is a bundle: %s", bundleImage, err)
	}
	if !isBundle {
		return Description{}, fmt.Errorf("Only bundles can be described, and %s is not a bundle", bundleImage)
	}

	allBundles, err := newBundle.FetchAllImagesRefs(opts.Concurrency, opts.Logger, artifactFetcher)
	if err != nil {
		return Description{}, fmt.Errorf("Retrieving Images from bundle: %s", err)
	}

	topBundle := refWithDescription{
		imgRef: bundle.NewBundleImageRef(lockconfig.ImageRef{Image: newBundle.DigestRef()}),
	}
	return topBundle.DescribeBundle(allBundles), nil
}

type refWithDescription struct {
	imgRef bundle.ImageRef
	bundle Description
}

func (r *refWithDescription) DescribeBundle(bundles []*bundle.Bundle) Description {
	var visitedImgs map[string]refWithDescription
	return r.describeBundleRec(visitedImgs, r.imgRef, bundles)
}

func (r *refWithDescription) describeBundleRec(visitedImgs map[string]refWithDescription, currentBundle bundle.ImageRef, bundles []*bundle.Bundle) Description {
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
			Content: Content{
				Bundles: map[string]Description{},
				Images:  map[string]ImageInfo{},
			},
		},
	}
	var newBundle *bundle.Bundle
	for _, b := range bundles {
		if b.DigestRef() == currentBundle.PrimaryLocation() {
			newBundle = b
			break
		}
	}
	if newBundle == nil {
		panic("Internal consistency: bundle could not be found in list of bundles")
	}

	imagesRefs := newBundle.ImagesRefs()
	sort.Slice(imagesRefs, func(i, j int) bool {
		return imagesRefs[i].Image < imagesRefs[j].Image
	})

	for _, ref := range imagesRefs {
		if ref.IsBundle == nil {
			panic("Internal consistency: IsBundle after processing must always have a value")
		}

		if *ref.IsBundle {
			bundleDesc := r.describeBundleRec(visitedImgs, ref, bundles)
			digest, err := name.NewDigest(bundleDesc.Image)
			if err != nil {
				panic(fmt.Sprintf("Internal inconsistency: image %s should be fully resolved", bundleDesc.Image))
			}
			desc.bundle.Content.Bundles[digest.DigestStr()] = bundleDesc
		} else {
			if ref.Error == "" {
				digest, err := name.NewDigest(ref.Image)
				if err != nil {
					panic(fmt.Sprintf("Internal inconsistency: image %s should be fully resolved", ref.Image))
				}
				desc.bundle.Content.Images[digest.DigestStr()] = ImageInfo{
					Image:       ref.PrimaryLocation(),
					Origin:      ref.Image,
					Annotations: ref.Annotations,
					ImageType:   ref.ImageType,
				}
			} else {
				desc.bundle.Content.Images[ref.Image] = ImageInfo{
					ImageType: ref.ImageType,
					Error:     ref.Error,
				}
			}
		}
	}

	return desc.bundle
}
