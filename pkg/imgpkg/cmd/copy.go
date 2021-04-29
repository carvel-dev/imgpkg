// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	ctlimgset "github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
	"github.com/k14s/imgpkg/pkg/imgpkg/registry"
	"github.com/spf13/cobra"
)

type CopyOptions struct {
	ImageFlags      ImageFlags
	BundleFlags     BundleFlags
	LockInputFlags  LockInputFlags
	LockOutputFlags LockOutputFlags
	TarFlags        TarFlags
	RegistryFlags   RegistryFlags

	RepoDst                 string
	Concurrency             int
	IncludeNonDistributable bool
}

func NewCopyOptions() *CopyOptions {
	return &CopyOptions{}
}

func NewCopyCmd(o *CopyOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copy",
		Short: "Copy a bundle from one location to another",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
		Example: `
    # Copy bundle dkalinin/app1-bundle to local tarball at /Volumes/app1-bundle.tar
    imgpkg copy -b dkalinin/app1-bundle --to-tar /Volumes/app1-bundle.tar

    # Copy bundle dkalinin/app1-bundle to another registry (or repository)
    imgpkg copy -b dkalinin/app1-bundle --to-repo internal-registry/app1-bundle

    # Copy image dkalinin/app1-image to another registry (or repository)
    imgpkg copy -i dkalinin/app1-image --to-repo internal-registry/app1-image`,
	}

	o.ImageFlags.SetCopy(cmd)
	o.BundleFlags.SetCopy(cmd)
	o.LockInputFlags.Set(cmd)
	o.LockOutputFlags.Set(cmd)
	o.TarFlags.Set(cmd)
	o.RegistryFlags.Set(cmd)
	cmd.Flags().StringVar(&o.RepoDst, "to-repo", "", "Location to upload assets")
	cmd.Flags().IntVar(&o.Concurrency, "concurrency", 5, "Concurrency")
	cmd.Flags().BoolVar(&o.IncludeNonDistributable, "include-non-distributable-layers", false,
		"Include non-distributable layers when copying an image/bundle")
	return cmd
}

func (o *CopyOptions) Run() error {
	if !o.hasOneSrc() {
		return fmt.Errorf("Expected either --lock, --bundle (-b), --image (-i), or --tar as a source")
	}
	if !o.hasOneDst() {
		return fmt.Errorf("Expected either --to-tar or --to-repo")
	}

	logger := ctlimg.NewLogger(os.Stderr)
	prefixedLogger := logger.NewPrefixedWriter("copy | ")

	registryOpts := o.RegistryFlags.AsRegistryOpts()
	registryOpts.IncludeNonDistributableLayers = o.IncludeNonDistributable

	registry, err := registry.NewRegistry(registryOpts)
	if err != nil {
		return fmt.Errorf("Unable to create a registry with the options %v: %v", registryOpts, err)
	}

	switch {
	case o.isTarSrc():
		if o.isTarDst() {
			return fmt.Errorf("Cannot use tar source (--tar) with tar destination (--to-tar)")
		}

		importRepo, err := regname.NewRepository(o.RepoDst)
		if err != nil {
			return fmt.Errorf("Building import repository ref: %s", err)
		}

		imageSet := ctlimgset.NewImageSet(o.Concurrency, prefixedLogger)
		tarImageSet := ctlimgset.NewTarImageSet(imageSet, o.Concurrency, prefixedLogger)

		processedImages, err := tarImageSet.Import(o.TarFlags.TarSrc, importRepo, registry)
		if err != nil {
			return err
		}

		informUserToUseTheNonDistributableFlagWithDescriptors(prefixedLogger, o.IncludeNonDistributable, processedImagesMediaType(processedImages))
		return o.writeLockOutput(processedImages, registry)

	case o.isRepoSrc():
		imageSet := ctlimgset.NewImageSet(o.Concurrency, prefixedLogger)

		repoSrc := CopyRepoSrc{
			logger:                  prefixedLogger,
			ImageFlags:              o.ImageFlags,
			BundleFlags:             o.BundleFlags,
			LockInputFlags:          o.LockInputFlags,
			IncludeNonDistributable: o.IncludeNonDistributable,

			registry:    registry,
			imageSet:    imageSet,
			tarImageSet: ctlimgset.NewTarImageSet(imageSet, o.Concurrency, prefixedLogger),
			Concurrency: o.Concurrency,
		}

		switch {
		case o.isTarDst():
			if o.LockOutputFlags.LockFilePath != "" {
				return fmt.Errorf("cannot output lock file with tar destination")
			}

			return repoSrc.CopyToTar(o.TarFlags.TarDst)

		case o.isRepoDst():
			processedImages, err := repoSrc.CopyToRepo(o.RepoDst)
			if err != nil {
				return err
			}

			return o.writeLockOutput(processedImages, registry)
		}
	}
	panic("Unreachable")
}

func (o *CopyOptions) writeLockOutput(processedImages *ctlimgset.ProcessedImages, registry registry.Registry) error {
	var foundBundle *bundle.Bundle
	for _, item := range processedImages.All() {
		plainImg := plainimage.NewFetchedPlainImageWithTag(item.DigestRef, item.UnprocessedImageRef.Tag, item.Image, item.ImageIndex)
		bundle := bundle.NewBundleFromPlainImage(plainImg, registry)

		ok, err := bundle.IsBundle()
		if err != nil {
			return fmt.Errorf("Check if '%s' is bundle: %s", item.DigestRef, err)
		}
		if ok {
			foundBundle = bundle
		}
	}

	if o.LockOutputFlags.LockFilePath != "" {
		if foundBundle != nil {
			return o.writeBundleLockOutput(foundBundle)
		}
		return o.writeImagesLockOutput(processedImages)
	}
	return nil
}

func (o *CopyOptions) isTarSrc() bool { return o.TarFlags.TarSrc != "" }

func (o *CopyOptions) isRepoSrc() bool {
	return o.ImageFlags.Image != "" || o.BundleFlags.Bundle != "" || o.LockInputFlags.LockFilePath != ""
}

func (o *CopyOptions) isTarDst() bool  { return o.TarFlags.TarDst != "" }
func (o *CopyOptions) isRepoDst() bool { return o.RepoDst != "" }

func (o *CopyOptions) hasOneDst() bool {
	repoSet := o.isRepoDst()
	tarSet := o.isTarDst()
	return (repoSet || tarSet) && !(repoSet && tarSet)
}

func (o *CopyOptions) hasOneSrc() bool {
	var seen bool
	for _, ref := range []string{o.LockInputFlags.LockFilePath, o.TarFlags.TarSrc,
		o.BundleFlags.Bundle, o.ImageFlags.Image} {
		if ref != "" {
			if seen {
				return false
			}
			seen = true
		}
	}
	return seen
}

func (o *CopyOptions) writeImagesLockOutput(processedImages *ctlimgset.ProcessedImages) error {
	imagesLock := lockconfig.ImagesLock{
		LockVersion: lockconfig.LockVersion{
			APIVersion: lockconfig.ImagesLockAPIVersion,
			Kind:       lockconfig.ImagesLockKind,
		},
	}

	if o.LockInputFlags.LockFilePath != "" {
		var err error
		imagesLock, err = lockconfig.NewImagesLockFromPath(o.LockInputFlags.LockFilePath)
		if err != nil {
			return err
		}
		for i, image := range imagesLock.Images {
			img, found := processedImages.FindByURL(ctlimgset.UnprocessedImageRef{DigestRef: image.Image})
			if !found {
				return fmt.Errorf("Expected image '%s' to have been copied but was not", image.Image)
			}
			imagesLock.Images[i].Image = img.DigestRef
		}
	} else {
		for _, img := range processedImages.All() {
			imagesLock.Images = append(imagesLock.Images, lockconfig.ImageRef{
				Image: img.DigestRef,
			})
		}
	}

	return imagesLock.WriteToPath(o.LockOutputFlags.LockFilePath)
}

func (o *CopyOptions) writeBundleLockOutput(bundle *bundle.Bundle) error {
	bundleLock := lockconfig.BundleLock{
		LockVersion: lockconfig.LockVersion{
			APIVersion: lockconfig.BundleLockAPIVersion,
			Kind:       lockconfig.BundleLockKind,
		},
		Bundle: lockconfig.BundleRef{
			Image: bundle.DigestRef(),
			Tag:   bundle.Tag(),
		},
	}

	return bundleLock.WriteToPath(o.LockOutputFlags.LockFilePath)
}

func processedImagesMediaType(processedImages *ctlimgset.ProcessedImages) []string {
	everyMediaType := []string{}
	for _, image := range processedImages.All() {
		if image.ImageIndex != nil {
			mediaTypes := everyMediaTypeForAnImageIndex(image.ImageIndex)
			everyMediaType = append(everyMediaType, mediaTypes...)
		} else if image.Image != nil {
			mediaTypes := everyMediaTypeForAnImage(image.Image)
			everyMediaType = append(everyMediaType, mediaTypes...)
		}
	}
	return everyMediaType
}

func everyMediaTypeForAnImageIndex(imageIndex regv1.ImageIndex) []string {
	everyMediaType := []string{}
	indexManifest, err := imageIndex.IndexManifest()
	if err != nil {
		return []string{}
	}
	for _, descriptor := range indexManifest.Manifests {
		if descriptor.MediaType.IsIndex() {
			imageIndex, err := imageIndex.ImageIndex(descriptor.Digest)
			if err != nil {
				continue
			}
			mediaTypesForImageIndex := everyMediaTypeForAnImageIndex(imageIndex)
			everyMediaType = append(everyMediaType, mediaTypesForImageIndex...)
		} else {
			image, err := imageIndex.Image(descriptor.Digest)
			if err != nil {
				continue
			}
			mediaTypeForImage := everyMediaTypeForAnImage(image)
			everyMediaType = append(everyMediaType, mediaTypeForImage...)
		}
	}
	return everyMediaType
}

func everyMediaTypeForAnImage(image regv1.Image) []string {
	var everyMediaType []string

	layers, err := image.Layers()
	if err != nil {
		return everyMediaType
	}

	for _, layer := range layers {
		mediaType, err := layer.MediaType()
		if err != nil {
			continue
		}
		everyMediaType = append(everyMediaType, string(mediaType))
	}
	return everyMediaType
}

func informUserToUseTheNonDistributableFlagWithDescriptors(logger *ctlimg.LoggerPrefixWriter, includeNonDistributableFlag bool, everyMediaType []string) {
	noNonDistributableLayers := true

	for _, mediaType := range everyMediaType {
		if !types.MediaType(mediaType).IsDistributable() {
			noNonDistributableLayers = false
		}
	}

	if includeNonDistributableFlag && noNonDistributableLayers {
		logger.WriteStr("Warning: '--include-non-distributable-layers' flag provided, but no images contained a non-distributable layer.")
	} else if !includeNonDistributableFlag && !noNonDistributableLayers {
		logger.WriteStr("Skipped layer due to it being non-distributable. If you would like to include non-distributable layers, use the --include-non-distributable-layers flag")
	}
}
