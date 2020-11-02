// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

type CopyOptions struct {
	ui ui.UI

	ImageFlags      ImageFlags
	BundleFlags     BundleFlags
	LockInputFlags  LockInputFlags
	LockOutputFlags LockOutputFlags
	TarFlags        TarFlags
	RegistryFlags   RegistryFlags

	RepoDst     string
	Concurrency int
}

func NewCopyOptions(ui ui.UI) *CopyOptions {
	return &CopyOptions{ui: ui}
}

func NewCopyCmd(o *CopyOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copy",
		Short: "Copy a bundle from one location to another",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
		Example: `
    # Copy bundle dkalinin/app1-bundle to local tarball at /Volumes/app1-bundle.tar
    imgpkg copy -b dkalinin/app1-bundle --to-tar /Volumes/app1-bundle.tar

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
	return cmd
}

func (o *CopyOptions) Run() error {
	if !o.hasOneSrc() {
		return fmt.Errorf("Expected either --lock, --bundle (-b), --image (-i), or --from-tar as a source")
	}

	if !o.hasOneDest() {
		return fmt.Errorf("Expected either --to-tar or --to-repo")
	}

	if o.isTarSrc() && o.isTarDst() {
		return fmt.Errorf("Cannot use tar src with tar dst")
	}

	logger := ctlimg.NewLogger(os.Stderr)
	prefixedLogger := logger.NewPrefixedWriter("copy | ")
	registry, err := ctlimg.NewRegistry(o.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return fmt.Errorf("Unable to create a registry with the options %v: %v", o.RegistryFlags.AsRegistryOpts(), err)
	}
	imageSet := ImageSet{o.Concurrency, prefixedLogger}

	var importRepo regname.Repository
	var unprocessedImageUrls *UnprocessedImageURLs
	var bundleURL string
	var processedImages *ProcessedImages
	switch {
	case o.isTarSrc():
		importRepo, err = regname.NewRepository(o.RepoDst)
		if err != nil {
			return fmt.Errorf("Building import repository ref: %s", err)
		}
		tarImageSet := TarImageSet{imageSet, o.Concurrency, prefixedLogger}
		processedImages, bundleURL, err = tarImageSet.Import(o.TarFlags.TarSrc, importRepo, registry)
	case o.isRepoSrc() && o.isTarDst():
		if o.LockOutputFlags.LockFilePath != "" {
			return fmt.Errorf("cannot output lock file with tar destination")
		}

		unprocessedImageUrls, bundleURL, err = o.GetUnprocessedImageURLs()
		if err != nil {
			return err
		}

		if bundleURL != "" {
			unprocessedImageUrls, err = checkBundleRepoForCollocatedImages(unprocessedImageUrls, bundleURL, registry)
			if err != nil {
				return err
			}
		}

		tarImageSet := TarImageSet{imageSet, o.Concurrency, prefixedLogger}
		err = tarImageSet.Export(unprocessedImageUrls, o.TarFlags.TarDst, registry) // download to tar
	case o.isRepoSrc() && o.isRepoDst():
		unprocessedImageUrls, bundleURL, err = o.GetUnprocessedImageURLs()
		if err != nil {
			return err
		}

		if bundleURL != "" {
			unprocessedImageUrls, err = checkBundleRepoForCollocatedImages(unprocessedImageUrls, bundleURL, registry)
			if err != nil {
				return err
			}
		}

		importRepo, err = regname.NewRepository(o.RepoDst)
		if err != nil {
			return fmt.Errorf("Building import repository ref: %s", err)
		}
		processedImages, err = imageSet.Relocate(unprocessedImageUrls, importRepo, registry)
	}

	if err != nil {
		return err
	}

	if o.LockOutputFlags.LockFilePath != "" {
		err = o.writeLockOutput(processedImages, bundleURL)
	}

	return err
}

func (o *CopyOptions) isTarSrc() bool {
	return o.TarFlags.TarSrc != ""
}

func (o *CopyOptions) isRepoSrc() bool {
	return o.ImageFlags.Image != "" || o.BundleFlags.Bundle != "" || o.LockInputFlags.LockFilePath != ""
}

func (o *CopyOptions) isTarDst() bool {
	return o.TarFlags.TarDst != ""
}

func (o *CopyOptions) isRepoDst() bool {
	return o.RepoDst != ""
}

func (o *CopyOptions) hasOneDest() bool {
	repoSet := o.isRepoDst()
	tarSet := o.isTarDst()
	return (repoSet || tarSet) && !(repoSet && tarSet)
}

func (o *CopyOptions) hasOneSrc() bool {
	var seen bool
	for _, ref := range []string{o.LockInputFlags.LockFilePath, o.TarFlags.TarSrc, o.BundleFlags.Bundle, o.ImageFlags.Image} {
		if ref != "" {
			if seen {
				return false
			}
			seen = true
		}
	}
	return seen
}

func (o *CopyOptions) GetUnprocessedImageURLs() (*UnprocessedImageURLs, string, error) {
	unprocessedImageURLs := NewUnprocessedImageURLs()
	var bundleRef string
	reg, err := image.NewRegistry(o.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return nil, "", fmt.Errorf("Unable to create a registry with the options %v: %v", o.RegistryFlags.AsRegistryOpts(), err)
	}
	switch {

	case o.LockInputFlags.LockFilePath != "":
		lock, err := ReadLockFile(o.LockInputFlags.LockFilePath)
		if err != nil {
			return nil, "", err
		}
		switch {
		case lock.Kind == "BundleLock":
			bundleLock, err := ReadBundleLockFile(o.LockInputFlags.LockFilePath)
			if err != nil {
				return nil, "", err
			}

			bundleRef = bundleLock.Spec.Image.DigestRef
			parsedRef, err := regname.ParseReference(bundleRef)
			if err != nil {
				return nil, "", err
			}

			img, err := reg.Image(parsedRef)
			if err != nil {
				return nil, "", err
			}

			isBundle, err := isBundle(img)
			if err != nil {
				return nil, "", err
			}

			if !isBundle {
				return nil, "", fmt.Errorf("Expected image flag when given an image reference. Please run with -i instead of -b, or use -b with a bundle reference")
			}

			images, err := GetReferencedImages(parsedRef, o.RegistryFlags.AsRegistryOpts())
			if err != nil {
				return nil, "", err
			}

			for _, image := range images {
				unprocessedImageURLs.Add(UnprocessedImageURL{URL: image.Image})
			}
			unprocessedImageURLs.Add(UnprocessedImageURL{URL: bundleRef, Tag: bundleLock.Spec.Image.OriginalTag})

		case lock.Kind == "ImagesLock":
			imgLock, err := ReadImageLockFile(o.LockInputFlags.LockFilePath)
			if err != nil {
				return nil, "", err
			}

			bundles, err := imgLock.CheckForBundles(reg)
			if err != nil {
				return nil, "", fmt.Errorf("Checking image lock for bundles: %s", err)
			}
			if len(bundles) != 0 {
				return nil, "", fmt.Errorf("Expected image lock to not contain bundle reference: '%v'", strings.Join(bundles, "', '"))
			}

			for _, img := range imgLock.Spec.Images {
				unprocessedImageURLs.Add(UnprocessedImageURL{URL: img.Image})
			}
		default:
			return nil, "", fmt.Errorf("Unexpected lock kind, expected bundleLock or imageLock, got: %v", lock.Kind)
		}

	case o.ImageFlags.Image != "":
		parsedRef, err := regname.ParseReference(o.ImageFlags.Image)
		if err != nil {
			return nil, "", err
		}

		var imageTag string
		if t, ok := parsedRef.(regname.Tag); ok {
			imageTag = t.TagStr()
		}

		img, err := reg.Image(parsedRef)
		if err != nil {
			return nil, "", err
		}

		digest, err := img.Digest()
		if err != nil {
			return nil, "", err
		}

		parsedRef, err = regname.NewDigest(fmt.Sprintf("%s@%s", parsedRef.Context().Name(), digest))
		if err != nil {
			return nil, "", err
		}

		isBundle, err := isBundle(img)
		if err != nil {
			return nil, "", err
		}

		if isBundle {
			return nil, "", fmt.Errorf("Expected bundle flag when copying a bundle, please use -b instead of -i")
		}

		unprocessedImageURLs.Add(UnprocessedImageURL{o.ImageFlags.Image, imageTag})

	default:
		bundleRef = o.BundleFlags.Bundle

		parsedRef, err := regname.ParseReference(bundleRef)
		if err != nil {
			return nil, "", err
		}

		var bundleTag string
		if t, ok := parsedRef.(regname.Tag); ok {
			bundleTag = t.TagStr()
		}

		img, err := reg.Image(parsedRef)
		if err != nil {
			return nil, "", err
		}

		digest, err := img.Digest()
		if err != nil {
			return nil, "", err
		}

		bundleRef = fmt.Sprintf("%s@%s", parsedRef.Context().Name(), digest)
		parsedRef, err = regname.NewDigest(bundleRef)
		if err != nil {
			return nil, "", err
		}

		isBundle, err := isBundle(img)
		if err != nil {
			return nil, "", err
		}

		if !isBundle {
			return nil, "", fmt.Errorf("Expected image flag when given an image reference. Please run with -i instead of -b, or use -b with a bundle reference")
		}

		images, err := GetReferencedImages(parsedRef, o.RegistryFlags.AsRegistryOpts())
		if err != nil {
			return nil, "", err
		}

		for _, img := range images {
			unprocessedImageURLs.Add(UnprocessedImageURL{URL: img.Image})
		}

		unprocessedImageURLs.Add(UnprocessedImageURL{URL: bundleRef, Tag: bundleTag})
	}

	return unprocessedImageURLs, bundleRef, nil
}

func (o *CopyOptions) writeLockOutput(processedImages *ProcessedImages, bundleURL string) error {

	var outBytes []byte
	var err error

	switch bundleURL {
	case "":
		iLock := ImageLock{ApiVersion: ImageLockAPIVersion, Kind: ImageLockKind}
		for _, img := range processedImages.All() {
			iLock.Spec.Images = append(
				iLock.Spec.Images,
				ImageDesc{
					Image: img.Image.URL,
				},
			)
		}

		outBytes, err = yaml.Marshal(iLock)
		if err != nil {
			return err
		}
	default:
		var originalTag, url string
		for _, img := range processedImages.All() {
			if img.UnprocessedImageURL.URL == bundleURL {
				originalTag = img.UnprocessedImageURL.Tag
				url = img.Image.URL
			}
		}

		if url == "" {
			return fmt.Errorf("could not find process item for url '%s'", bundleURL)
		}

		bLock := BundleLock{
			ApiVersion: BundleLockAPIVersion,
			Kind:       BundleLockKind,
			Spec:       BundleSpec{Image: ImageLocation{DigestRef: url, OriginalTag: originalTag}},
		}
		outBytes, err = yaml.Marshal(bLock)
		if err != nil {
			return err
		}

	}

	return ioutil.WriteFile(o.LockOutputFlags.LockFilePath, outBytes, 0700)
}

func checkBundleRepoForCollocatedImages(foundImages *UnprocessedImageURLs, bundleURL string, registry ctlimg.Registry) (*UnprocessedImageURLs, error) {
	checkedURLs := NewUnprocessedImageURLs()
	bundleRef, err := regname.ParseReference(bundleURL)
	if err != nil {
		return nil, err
	}
	bundleRepo := bundleRef.Context().Name()

	for _, img := range foundImages.All() {
		if img.URL == bundleURL {
			checkedURLs.Add(img)
			continue
		}

		newURL, err := ImageWithRepository(img.URL, bundleRepo)
		if err != nil {
			return nil, err
		}
		ref, err := regname.NewDigest(newURL, regname.StrictValidation)
		if err != nil {
			return nil, err
		}

		_, err = registry.Generic(ref)
		if err == nil {
			checkedURLs.Add(UnprocessedImageURL{newURL, img.Tag})
		} else {
			checkedURLs.Add(img)
		}
	}

	return checkedURLs, nil
}
