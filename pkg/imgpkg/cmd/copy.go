// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	ctlimgset "github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
	"github.com/k14s/imgpkg/pkg/imgpkg/registry"
	"github.com/k14s/imgpkg/pkg/imgpkg/signature"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
	"github.com/spf13/cobra"
)

const rootBundleLabelKey string = "dev.carvel.imgpkg.copy.root-bundle"

type CopyOptions struct {
	ImageFlags      ImageFlags
	BundleFlags     BundleFlags
	LockInputFlags  LockInputFlags
	LockOutputFlags LockOutputFlags
	TarFlags        TarFlags
	RegistryFlags   RegistryFlags
	SignatureFlags  SignatureFlags

	RepoDst string

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
	o.SignatureFlags.Set(cmd)
	cmd.Flags().StringVar(&o.RepoDst, "to-repo", "", "Location to upload assets")
	cmd.Flags().IntVar(&o.Concurrency, "concurrency", 5, "Concurrency")
	cmd.Flags().BoolVar(&o.IncludeNonDistributable, "include-non-distributable-layers", false,
		"Include non-distributable layers when copying an image/bundle")
	return cmd
}

func (c *CopyOptions) Run() error {
	if !c.hasOneSrc() {
		return fmt.Errorf("Expected either --lock, --bundle (-b), --image (-i), or --tar as a source")
	}
	if !c.hasOneDst() {
		return fmt.Errorf("Expected either --to-tar or --to-repo")
	}

	registryOpts := c.RegistryFlags.AsRegistryOpts()
	registryOpts.IncludeNonDistributableLayers = c.IncludeNonDistributable

	reg, err := registry.NewRegistry(registryOpts)
	if err != nil {
		return fmt.Errorf("Unable to create a registry with the options %v: %v", registryOpts, err)
	}

	logger := util.NewLogger(os.Stderr)
	prefixedLogger := logger.NewPrefixedWriter("copy | ")
	levelLogger := logger.NewLevelLogger(util.LogWarn, prefixedLogger)

	imagesUploaderLogger := logger.NewProgressBar(levelLogger, "done uploading images", "Error uploading images")
	regWithProgress := registry.NewRegistryWithProgress(reg, imagesUploaderLogger)

	imageSet := ctlimgset.NewImageSet(c.Concurrency, prefixedLogger)

	var signatureRetriever SignatureRetriever
	if c.SignatureFlags.CopyCosignSignatures {
		signatureRetriever = signature.NewSignatures(signature.NewCosign(reg), c.Concurrency)
	} else {
		signatureRetriever = signature.NewNoop()
	}

	repoSrc := CopyRepoSrc{
		ImageFlags:              c.ImageFlags,
		BundleFlags:             c.BundleFlags,
		LockInputFlags:          c.LockInputFlags,
		TarFlags:                c.TarFlags,
		IncludeNonDistributable: c.IncludeNonDistributable,

		logger:             levelLogger,
		registry:           regWithProgress,
		imageSet:           imageSet,
		tarImageSet:        ctlimgset.NewTarImageSet(imageSet, c.Concurrency, prefixedLogger),
		Concurrency:        c.Concurrency,
		signatureRetriever: signatureRetriever,
	}

	switch {
	case c.TarFlags.IsDst():
		if c.TarFlags.IsSrc() {
			return fmt.Errorf("Cannot use tar source (--tar) with tar destination (--to-tar)")
		}
		if c.LockOutputFlags.LockFilePath != "" {
			return fmt.Errorf("Cannot output lock file with tar destination")
		}
		return repoSrc.CopyToTar(c.TarFlags.TarDst)

	case c.isRepoDst():
		processedImages, err := repoSrc.CopyToRepo(c.RepoDst)
		if err != nil {
			return err
		}
		return c.writeLockOutput(processedImages, reg)

	default:
		panic("Unreachable")
	}
}

func (c *CopyOptions) writeLockOutput(processedImages *ctlimgset.ProcessedImages, registry registry.Registry) error {
	if c.LockOutputFlags.LockFilePath == "" {
		return nil
	}

	processedImageRootBundle := c.findProcessedImageRootBundle(processedImages)

	if processedImageRootBundle != nil {
		// this is an optimization to avoid getting an image descriptor for an ImageIndex, since we know
		// it definetely will not be a bundle
		if processedImageRootBundle.ImageIndex != nil {
			panic(fmt.Errorf("Internal inconsistency: '%s' should be a bundle but it is not", processedImageRootBundle.DigestRef))
		}

		plainImg := plainimage.NewFetchedPlainImageWithTag(processedImageRootBundle.DigestRef, processedImageRootBundle.UnprocessedImageRef.Tag, processedImageRootBundle.Image)
		foundBundle := bundle.NewBundleFromPlainImage(plainImg, registry)
		ok, err := foundBundle.IsBundle()
		if err != nil {
			return fmt.Errorf("Check if '%s' is bundle: %s", processedImageRootBundle.DigestRef, err)
		}
		if !ok {
			panic(fmt.Errorf("Internal inconsistency: '%s' should be a bundle but it is not", processedImageRootBundle.DigestRef))
		}

		return c.writeBundleLockOutput(foundBundle)
	}

	// if the tarball was created with an older version (prior to assign a label to the root bundle) and it contains a bundle
	// then return an error to the user informing them to recreate the tarball, since we don't know which is the root bundle.
	err := c.informUserIfTarballNeedsToBeRecreated(processedImages, registry)
	if err != nil {
		return err
	}

	return c.writeImagesLockOutput(processedImages)
}

func (c *CopyOptions) findProcessedImageRootBundle(processedImages *ctlimgset.ProcessedImages) *ctlimgset.ProcessedImage {
	var bundleProcessedImage *ctlimgset.ProcessedImage

	for _, processedImage := range processedImages.All() {
		if _, ok := processedImage.Labels[rootBundleLabelKey]; ok {
			if bundleProcessedImage != nil {
				panic("Internal inconsistency: expected only 1 root bundle")
			}
			bundleProcessedImage = &ctlimgset.ProcessedImage{
				UnprocessedImageRef: processedImage.UnprocessedImageRef,
				DigestRef:           processedImage.DigestRef,
				Image:               processedImage.Image,
				ImageIndex:          processedImage.ImageIndex,
			}
		}
	}
	return bundleProcessedImage
}

func (c *CopyOptions) informUserIfTarballNeedsToBeRecreated(processedImages *ctlimgset.ProcessedImages, registry registry.Registry) error {
	for _, item := range processedImages.All() {
		// this is an optimization to avoid getting an image descriptor for an ImageIndex, since we know
		// it definetely will not be a bundle
		if item.ImageIndex != nil {
			continue
		}

		plainImg := plainimage.NewFetchedPlainImageWithTag(item.DigestRef, item.UnprocessedImageRef.Tag, item.Image)
		bundle := bundle.NewBundleFromPlainImage(plainImg, registry)

		ok, err := bundle.IsBundle()
		if err != nil {
			return fmt.Errorf("Check if '%s' is bundle: %s", item.DigestRef, err)
		}
		if ok {
			return fmt.Errorf("Unable to determine correct root bundle to use for lock-output. hint: if copying from a tarball, try re-generating the tarball")
		}
	}
	return nil
}

func (c *CopyOptions) isRepoDst() bool { return c.RepoDst != "" }

func (c *CopyOptions) hasOneDst() bool {
	repoSet := c.isRepoDst()
	tarSet := c.TarFlags.IsDst()
	return (repoSet || tarSet) && !(repoSet && tarSet)
}

func (c *CopyOptions) hasOneSrc() bool {
	var seen bool
	for _, ref := range []string{c.LockInputFlags.LockFilePath, c.TarFlags.TarSrc,
		c.BundleFlags.Bundle, c.ImageFlags.Image} {
		if ref != "" {
			if seen {
				return false
			}
			seen = true
		}
	}
	return seen
}

func (c *CopyOptions) writeImagesLockOutput(processedImages *ctlimgset.ProcessedImages) error {
	imagesLock := lockconfig.ImagesLock{
		LockVersion: lockconfig.LockVersion{
			APIVersion: lockconfig.ImagesLockAPIVersion,
			Kind:       lockconfig.ImagesLockKind,
		},
	}

	if c.LockInputFlags.LockFilePath != "" {
		var err error
		imagesLock, err = lockconfig.NewImagesLockFromPath(c.LockInputFlags.LockFilePath)
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

	return imagesLock.WriteToPath(c.LockOutputFlags.LockFilePath)
}

func (c *CopyOptions) writeBundleLockOutput(bundle *bundle.Bundle) error {
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

	return bundleLock.WriteToPath(c.LockOutputFlags.LockFilePath)
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

func informUserToUseTheNonDistributableFlagWithDescriptors(logger util.LoggerWithLevels, includeNonDistributableFlag bool, everyMediaType []string) {
	noNonDistributableLayers := true

	for _, mediaType := range everyMediaType {
		if !types.MediaType(mediaType).IsDistributable() {
			noNonDistributableLayers = false
		}
	}

	if includeNonDistributableFlag && noNonDistributableLayers {
		logger.Warnf("'--include-non-distributable-layers' flag provided, but no images contained a non-distributable layer.")
	} else if !includeNonDistributableFlag && !noNonDistributableLayers {
		logger.Warnf("Skipped layer due to it being non-distributable. If you would like to include non-distributable layers, use the --include-non-distributable-layers flag")
	}
}
