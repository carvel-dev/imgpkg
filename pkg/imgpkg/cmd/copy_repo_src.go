// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagelayers"

	regname "github.com/google/go-containerregistry/pkg/name"
	ctlbundle "github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	ctlimgset "github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
)

type CopyRepoSrc struct {
	ImageFlags                  ImageFlags
	BundleFlags                 BundleFlags
	LockInputFlags              LockInputFlags
	IncludeNonDistributableFlag IncludeNonDistributableFlag
	logger                      *ctlimg.LoggerPrefixWriter
	imageSet                    ctlimgset.ImageSet
	tarImageSet                 ctlimgset.TarImageSet
	registry                    ctlimg.ImagesReaderWriter
}

func (o CopyRepoSrc) CopyToTar(dstPath string) error {
	unprocessedImageRefs, err := o.getSourceImages()
	if err != nil {
		return err
	}

	ids, err := o.imageSet.Export(unprocessedImageRefs, o.registry)
	if err != nil {
		return err
	}

	defer func() {
		warnIfIncludeNonDistributableFlagWasProvidedButNoneOfTheLayersWereNonDistributable(o.logger, o.IncludeNonDistributableFlag.IncludeNonDistributable, ids.Descriptors())
	}()

	return o.tarImageSet.Export(ids, dstPath, imagelayers.NewImageLayerWriterCheck(o.IncludeNonDistributableFlag.IncludeNonDistributable))
}

func (o CopyRepoSrc) CopyToRepo(repo string) (*ctlimgset.ProcessedImages, error) {
	unprocessedImageRefs, err := o.getSourceImages()
	if err != nil {
		return nil, err
	}

	importRepo, err := regname.NewRepository(repo)
	if err != nil {
		return nil, fmt.Errorf("Building import repository ref: %s", err)
	}

	processedImages, err := o.imageSet.Relocate(unprocessedImageRefs, importRepo, o.registry)
	if err != nil {
		return nil, err
	}

	return processedImages, nil
}

func (o CopyRepoSrc) getSourceImages() (*ctlimgset.UnprocessedImageRefs, error) {
	unprocessedImageRefs := ctlimgset.NewUnprocessedImageRefs()

	switch {
	case o.LockInputFlags.LockFilePath != "":
		bundleLock, imagesLock, err := lockconfig.NewLockFromPath(o.LockInputFlags.LockFilePath)
		if err != nil {
			return nil, err
		}

		switch {
		case bundleLock != nil:
			bundle := ctlbundle.NewBundle(bundleLock.Bundle.Image, o.registry)

			imagesLock, err := bundle.ImagesLockLocalized()
			if err != nil {
				if ctlbundle.IsNotBundleError(err) {
					return nil, fmt.Errorf("Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
				}
				return nil, err
			}

			for _, img := range imagesLock.Images {
				unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: img.Image})
			}

			unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{
				DigestRef: bundleLock.Bundle.Image,
				Tag:       bundleLock.Bundle.Tag,
			})

			return unprocessedImageRefs, nil

		case imagesLock != nil:
			for _, img := range imagesLock.Images {
				plainImg := plainimage.NewPlainImage(img.Image, o.registry)

				ok, err := ctlbundle.NewBundleFromPlainImage(plainImg, o.registry).IsBundle()
				if err != nil {
					return nil, err
				}
				if ok {
					return nil, fmt.Errorf("Expected bundle flag when copying a bundle (hint: Use -b instead of -i for bundles)")
				}
				unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: plainImg.DigestRef()})
			}
			return unprocessedImageRefs, nil

		default:
			panic("Unreachable")
		}

	case o.ImageFlags.Image != "":
		plainImg := plainimage.NewPlainImage(o.ImageFlags.Image, o.registry)

		ok, err := ctlbundle.NewBundleFromPlainImage(plainImg, o.registry).IsBundle()
		if err != nil {
			return nil, err
		}
		if ok {
			return nil, fmt.Errorf("Expected bundle flag when copying a bundle (hint: Use -b instead of -i for bundles)")
		}

		unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: plainImg.DigestRef()})
		return unprocessedImageRefs, nil

	default:
		bundle := ctlbundle.NewBundle(o.BundleFlags.Bundle, o.registry)

		// TODO switch to using fallback URLs for each image
		// instead of trying to use localized bundle URLs here
		imagesLock, err := bundle.ImagesLockLocalized()
		if err != nil {
			if ctlbundle.IsNotBundleError(err) {
				return nil, fmt.Errorf("Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
			}
			return nil, err
		}

		for _, img := range imagesLock.Images {
			unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: img.Image})
		}

		unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: bundle.DigestRef(), Tag: bundle.Tag()})

		return unprocessedImageRefs, nil
	}

	panic("Unreachable")
}

func warnIfIncludeNonDistributableFlagWasProvidedButNoneOfTheLayersWereNonDistributable(logger *ctlimg.LoggerPrefixWriter, includeNonDistributable bool, descriptors []imagedesc.ImageOrImageIndexDescriptor) {
	if !includeNonDistributable {
		return
	}

	noNonDistributableLayers := true
	for _, td := range descriptors {
		for _, layer := range td.Image.Layers {
			if !layer.IsDistributable() {
				noNonDistributableLayers = false
			}
		}
	}

	if noNonDistributableLayers {
		logger.WriteStr("Warning: '--include-non-distributable' flag provided, but no images contained a non-distributable layer.")
	}
}