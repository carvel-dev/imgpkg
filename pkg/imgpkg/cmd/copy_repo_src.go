// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	regname "github.com/google/go-containerregistry/pkg/name"
	ctlbundle "github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	ctlimgset "github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
)

type SignatureRetriever interface {
	Fetch(images *imageset.UnprocessedImageRefs) (*imageset.UnprocessedImageRefs, error)
}

type CopyRepoSrc struct {
	ImageFlags              ImageFlags
	BundleFlags             BundleFlags
	LockInputFlags          LockInputFlags
	IncludeNonDistributable bool
	Concurrency             int
	logger                  util.LoggerWithLevels
	imageSet                ctlimgset.ImageSet
	tarImageSet             ctlimgset.TarImageSet
	registry                ctlimgset.ImagesReaderWriter
	signatureRetriever      SignatureRetriever
}

func (c CopyRepoSrc) CopyToTar(dstPath string) error {
	unprocessedImageRefs, _, err := c.getSourceImages()
	if err != nil {
		return err
	}

	signatures, err := c.signatureRetriever.Fetch(unprocessedImageRefs)
	if err != nil {
		return err
	}

	for _, signature := range signatures.All() {
		unprocessedImageRefs.Add(signature)
	}

	ids, err := c.tarImageSet.Export(unprocessedImageRefs, dstPath, c.registry, imagetar.NewImageLayerWriterCheck(c.IncludeNonDistributable))
	if err != nil {
		return err
	}

	informUserToUseTheNonDistributableFlagWithDescriptors(c.logger, c.IncludeNonDistributable, imageRefDescriptorsMediaTypes(ids))

	return nil
}

func (c *CopyRepoSrc) CopyToRepo(repo string) (*ctlimgset.ProcessedImages, error) {
	c.logger.Tracef(fmt.Sprintf("CopyToRepo(%s)\n", repo))
	unprocessedImageRefs, bundles, err := c.getSourceImages()
	if err != nil {
		return nil, err
	}

	c.logger.Debugf("fetching signatures\n")
	signatures, err := c.signatureRetriever.Fetch(unprocessedImageRefs)
	if err != nil {
		return nil, err
	}
	for _, signature := range signatures.All() {
		unprocessedImageRefs.Add(signature)
	}

	importRepo, err := regname.NewRepository(repo)
	if err != nil {
		return nil, fmt.Errorf("Building import repository ref: %s", err)
	}

	c.logger.Debugf("copy the fetched images\n")
	processedImages, ids, err := c.imageSet.Relocate(unprocessedImageRefs, importRepo, c.registry)
	if err != nil {
		return nil, err
	}

	for _, bundle := range bundles {
		if err := bundle.NoteCopy(processedImages, c.registry, c.logger); err != nil {
			return nil, fmt.Errorf("Creating copy information for bundle %s: %s", bundle.DigestRef(), err)
		}
	}
	informUserToUseTheNonDistributableFlagWithDescriptors(c.logger, c.IncludeNonDistributable, imageRefDescriptorsMediaTypes(ids))

	return processedImages, nil
}

func (c *CopyRepoSrc) getSourceImages() (*ctlimgset.UnprocessedImageRefs, []*ctlbundle.Bundle, error) {
	unprocessedImageRefs := ctlimgset.NewUnprocessedImageRefs()

	switch {
	case c.LockInputFlags.LockFilePath != "":
		bundleLock, imagesLock, err := lockconfig.NewLockFromPath(c.LockInputFlags.LockFilePath)
		if err != nil {
			return nil, nil, err
		}

		switch {
		case bundleLock != nil:
			c.logger.Tracef("get images from BundleLock file\n")
			_, bundles, imagesRef, err := c.getBundleImageRefs(bundleLock.Bundle.Image)
			if err != nil {
				return nil, nil, err
			}

			for _, img := range imagesRef {
				unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: img.PrimaryLocation()})
			}

			unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{
				DigestRef: bundleLock.Bundle.Image,
				Tag:       bundleLock.Bundle.Tag,
			})

			return unprocessedImageRefs, bundles, nil

		case imagesLock != nil:
			c.logger.Tracef("get images from ImagesLock file\n")
			for _, img := range imagesLock.Images {
				plainImg := plainimage.NewPlainImage(img.Image, c.registry)

				ok, err := ctlbundle.NewBundleFromPlainImage(plainImg, c.registry).IsBundle()
				if err != nil {
					return nil, nil, err
				}
				if ok {
					return nil, nil, fmt.Errorf("Unable to copy bundles using an Images Lock file (hint: Create a bundle with these images)")
				}

				unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: plainImg.DigestRef()})
			}
			return unprocessedImageRefs, nil, nil

		default:
			panic("Unreachable")
		}

	case c.ImageFlags.Image != "":
		c.logger.Tracef("copy single image\n")
		plainImg := plainimage.NewPlainImage(c.ImageFlags.Image, c.registry)

		ok, err := ctlbundle.NewBundleFromPlainImage(plainImg, c.registry).IsBundle()
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return nil, nil, fmt.Errorf("Expected bundle flag when copying a bundle (hint: Use -b instead of -i for bundles)")
		}

		unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: plainImg.DigestRef(), Tag: plainImg.Tag()})
		return unprocessedImageRefs, nil, nil

	default:
		c.logger.Tracef("copy bundle\n")
		bundle, allBundles, imagesRef, err := c.getBundleImageRefs(c.BundleFlags.Bundle)
		if err != nil {
			return nil, nil, err
		}

		for _, img := range imagesRef {
			unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: img.PrimaryLocation()})
		}

		unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: bundle.DigestRef(), Tag: bundle.Tag()})

		return unprocessedImageRefs, allBundles, nil
	}
}

func (c *CopyRepoSrc) getBundleImageRefs(bundleRef string) (*ctlbundle.Bundle, []*ctlbundle.Bundle, ctlbundle.ImagesRef, error) {
	bundle := ctlbundle.NewBundle(bundleRef, c.registry)
	isBundle, err := bundle.IsBundle()
	if err != nil {
		return nil, nil, ctlbundle.ImagesRef{}, err
	}
	if !isBundle {
		return nil, nil, ctlbundle.ImagesRef{}, fmt.Errorf("Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
	}

	nestedBundles, imageRefs, err := bundle.AllImagesRefs(c.Concurrency, c.logger)
	if err != nil {
		return nil, nil, ctlbundle.ImagesRef{}, fmt.Errorf("Reading Images from Bundle: %s", err)
	}
	return bundle, nestedBundles, imageRefs, nil
}

func imageRefDescriptorsMediaTypes(ids *imagedesc.ImageRefDescriptors) []string {
	mediaTypes := []string{}
	for _, descriptor := range ids.Descriptors() {
		if descriptor.Image != nil {
			for _, layerDescriptor := range (*descriptor.Image).Layers {
				mediaTypes = append(mediaTypes, layerDescriptor.MediaType)
			}
		}

	}
	return mediaTypes
}
