// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	lf "github.com/k14s/imgpkg/pkg/imgpkg/lockfiles"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var _ lf.ImageRetriever = ctlimg.Registry{}

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
		return fmt.Errorf("Cannot use tar source (--from-tar) with tar destination (--to-tar)")
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
	var bundleRef regname.Reference
	var processedImages *ProcessedImages
	switch {
	case o.isTarSrc():
		importRepo, err = regname.NewRepository(o.RepoDst)
		if err != nil {
			return fmt.Errorf("Building import repository ref: %s", err)
		}
		tarImageSet := TarImageSet{imageSet, o.Concurrency, prefixedLogger}

		//TODO: Make Import Return a Reference instead of an URL
		var bundleURL string
		processedImages, bundleURL, err = tarImageSet.Import(o.TarFlags.TarSrc, importRepo, registry)
		if bundleURL != "" {
			bundleRef, err = regname.ParseReference(bundleURL)
			if err != nil {
				return err
			}
		}
	case o.isRepoSrc() && o.isTarDst():
		if o.LockOutputFlags.LockFilePath != "" {
			return fmt.Errorf("cannot output lock file with tar destination")
		}

		unprocessedImageUrls, bundleRef, err = o.GetUnprocessedImageURLs()
		if err != nil {
			return err
		}

		if bundleRef != nil {
			unprocessedImageUrls, err = checkBundleRepoForCollocatedImages(unprocessedImageUrls, bundleRef, registry)
			if err != nil {
				return err
			}
		}

		tarImageSet := TarImageSet{imageSet, o.Concurrency, prefixedLogger}
		err = tarImageSet.Export(unprocessedImageUrls, o.TarFlags.TarDst, registry) // download to tar
	case o.isRepoSrc() && o.isRepoDst():
		unprocessedImageUrls, bundleRef, err = o.GetUnprocessedImageURLs()
		if err != nil {
			return err
		}

		if bundleRef != nil {
			unprocessedImageUrls, err = checkBundleRepoForCollocatedImages(unprocessedImageUrls, bundleRef, registry)
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
		err = o.writeLockOutput(processedImages, bundleRef)
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

func (o *CopyOptions) GetUnprocessedImageURLs() (*UnprocessedImageURLs, regname.Reference, error) {
	unprocessedImageURLs := NewUnprocessedImageURLs()
	var bundleRef regname.Reference
	reg, err := image.NewRegistry(o.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to create a registry with the options %v: %v", o.RegistryFlags.AsRegistryOpts(), err)
	}
	switch {
	case o.LockInputFlags.LockFilePath != "":
		lock, err := lf.ReadLockFile(o.LockInputFlags.LockFilePath)
		if err != nil {
			return nil, nil, err
		}

		var lockFileImages []regname.Reference
		switch {
		case lock.Kind == lf.BundleLockKind:
			var bundleOriginalTag string
			bundleRef, bundleOriginalTag, lockFileImages, err = lf.CollectBundleURLs(o.LockInputFlags.LockFilePath, reg)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "Unable to verify OCI Image references present on Bundle Lock file")
			}

			unprocessedImageURLs.Add(UnprocessedImageURL{URL: bundleRef.Name(), Tag: bundleOriginalTag})
		case lock.Kind == lf.ImagesLockKind:
			lockFileImages, err = lf.CollectImageLockURLs(o.LockInputFlags.LockFilePath, reg)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "Unable to verify OCI Image references present on Image Lock file")
			}
		default:
			return nil, nil, fmt.Errorf("Unexpected lock kind. Expected BundleLock or ImagesLock, got: %v", lock.Kind)
		}

		for _, ref := range lockFileImages {
			unprocessedImageURLs.Add(UnprocessedImageURL{URL: ref.Name()})
		}

	case o.ImageFlags.Image != "":
		parsedRef, img, err := getRefAndImage(o.ImageFlags.Image, reg)
		if err != nil {
			return nil, nil, err
		}

		if err := checkIfBundle(img, false, fmt.Errorf("Expected bundle flag when copying a bundle, please use -b instead of -i")); err != nil {
			return nil, nil, err
		}

		imageTag := getTag(parsedRef)
		unprocessedImageURLs.Add(UnprocessedImageURL{o.ImageFlags.Image, imageTag})

	default:
		var img regv1.Image
		bundleRef, img, err = getRefAndImage(o.BundleFlags.Bundle, reg)
		if err != nil {
			return nil, nil, err
		}

		bundleTag := getTag(bundleRef)
		refWithDigest, err := getRefWithDigest(bundleRef, img)
		if err != nil {
			return nil, nil, err
		}

		if err := checkIfBundle(img, true, fmt.Errorf("Expected image flag when given an image reference. Please run with -i instead of -b, or use -b with a bundle reference")); err != nil {
			return nil, nil, err
		}

		images, err := lf.GetReferencedImages(refWithDigest, reg)
		if err != nil {
			return nil, nil, err
		}

		for _, img := range images {
			unprocessedImageURLs.Add(UnprocessedImageURL{URL: img.Image})
		}
		unprocessedImageURLs.Add(UnprocessedImageURL{URL: bundleRef.Name(), Tag: bundleTag})
	}

	return unprocessedImageURLs, bundleRef, nil
}

// Get the parsed image reference and associated image struct from a registry
func getRefAndImage(ref string, reg image.Registry) (regname.Reference, regv1.Image, error) {
	parsedRef, err := regname.ParseReference(ref)
	if err != nil {
		return nil, nil, err
	}

	img, err := reg.Image(parsedRef)
	if err != nil {
		return nil, nil, err
	}

	return parsedRef, img, err
}

// Get image reference with digest
func getRefWithDigest(parsedRef regname.Reference, img regv1.Image) (regname.Reference, error) {
	digest, err := img.Digest()
	if err != nil {
		return nil, err
	}
	refWithDigest, err := regname.NewDigest(fmt.Sprintf("%s@%s", parsedRef.Context().Name(), digest))
	if err != nil {
		return nil, err
	}
	return refWithDigest, err
}

// Get the tag from an image reference. Returns empty string
// if no tag found.
func getTag(parsedRef regname.Reference) string {
	var tag string
	if t, ok := parsedRef.(regname.Tag); ok {
		tag = t.TagStr()
	}
	return tag
}

// Determine whether an image is a Bundle or is not a Bundle
func checkIfBundle(img regv1.Image, expectsBundle bool, errMsg error) error {
	isBundle, err := lf.IsBundle(img)
	if err != nil {
		return err
	}
	// bundleCheck lets function caller determine whether to err
	// on if img is a Bundle or is not
	if isBundle != expectsBundle {
		// errMsg is custom err message if isBundle != expectsBundle
		// that caller can specify
		return errMsg
	}

	return nil
}

func (o *CopyOptions) writeLockOutput(processedImages *ProcessedImages, bundleURL regname.Reference) error {
	var outBytes []byte
	var err error

	switch bundleURL {
	case nil:
		iLock := lf.ImageLock{ApiVersion: lf.ImagesLockAPIVersion, Kind: lf.ImagesLockKind}
		for _, img := range processedImages.All() {
			iLock.Spec.Images = append(
				iLock.Spec.Images,
				lf.ImageDesc{
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
			if img.UnprocessedImageURL.URL == bundleURL.Name() {
				originalTag = img.UnprocessedImageURL.Tag
				url = img.Image.URL
			}
		}

		if url == "" {
			return fmt.Errorf("could not find process item for url '%s'", bundleURL)
		}

		bLock := lf.BundleLock{
			ApiVersion: lf.BundleLockAPIVersion,
			Kind:       lf.BundleLockKind,
			Spec:       lf.BundleSpec{Image: lf.ImageLocation{DigestRef: url, OriginalTag: originalTag}},
		}
		outBytes, err = yaml.Marshal(bLock)
		if err != nil {
			return err
		}

	}

	return ioutil.WriteFile(o.LockOutputFlags.LockFilePath, outBytes, 0700)
}

func checkBundleRepoForCollocatedImages(foundImages *UnprocessedImageURLs, bundleRef regname.Reference, registry ctlimg.Registry) (*UnprocessedImageURLs, error) {
	checkedURLs := NewUnprocessedImageURLs()
	bundleRepo := bundleRef.Context().Name()

	for _, img := range foundImages.All() {
		if img.URL == bundleRef.Name() {
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
