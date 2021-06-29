// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	goui "github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	plainimg "github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
)

const (
	BundleConfigLabel = "dev.carvel.imgpkg.bundle"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesLockReader
type ImagesLockReader interface {
	Read(img regv1.Image) (lockconfig.ImagesLock, error)
}

type Bundle struct {
	plainImg         *plainimg.PlainImage
	imgRetriever     ctlimg.ImagesMetadata
	imagesLockReader ImagesLockReader
	imagesRef        map[string]ImageRef
}

func NewBundle(ref string, imagesMetadata ctlimg.ImagesMetadata) *Bundle {
	return NewBundleWithReader(ref, imagesMetadata, &singleLayerReader{})
}

func NewBundleFromPlainImage(plainImg *plainimg.PlainImage, imagesMetadata ctlimg.ImagesMetadata) *Bundle {
	return &Bundle{plainImg: plainImg, imgRetriever: imagesMetadata, imagesLockReader: &singleLayerReader{}, imagesRef: map[string]ImageRef{}}
}

func NewBundleWithReader(ref string, imagesMetadata ctlimg.ImagesMetadata, imagesLockReader ImagesLockReader) *Bundle {
	return &Bundle{plainImg: plainimg.NewPlainImage(ref, imagesMetadata), imgRetriever: imagesMetadata, imagesLockReader: imagesLockReader, imagesRef: map[string]ImageRef{}}
}

func (o *Bundle) DigestRef() string { return o.plainImg.DigestRef() }
func (o *Bundle) Repo() string      { return o.plainImg.Repo() }
func (o *Bundle) Tag() string       { return o.plainImg.Tag() }

func (o *Bundle) addImageRefs(refs ...ImageRef) {
	for _, imageRef := range refs {
		o.imagesRef[imageRef.Image] = imageRef
	}
}

func (o *Bundle) imageRef(imageDigest string) (ImageRef, bool) {
	ref, found := o.imagesRef[imageDigest]
	if !found {
		for _, imgRef := range o.imagesRef {
			for _, loc := range imgRef.Locations() {
				if loc == imageDigest {
					return imgRef, true
				}
			}
		}
	}
	return ref, found
}

func (o *Bundle) imageRefs() []ImageRef {
	var imgsRef []ImageRef
	for _, ref := range o.imagesRef {
		imgsRef = append(imgsRef, ref)
	}
	return imgsRef
}

func (o *Bundle) NoteCopy(processedImages *imageset.ProcessedImages, reg ImagesMetadataWriter, logger util.LoggerWithLevels) error {
	locationsCfg := ImageLocationsConfig{
		APIVersion: LocationAPIVersion,
		Kind:       ImageLocationsKind,
	}
	var bundleProcessedImage imageset.ProcessedImage
	for _, image := range processedImages.All() {
		ref, found := o.imageRef(image.UnprocessedImageRef.DigestRef)
		if found {
			locationsCfg.Images = append(locationsCfg.Images, ImageLocation{
				Image:    ref.Image,
				IsBundle: *ref.IsBundle,
			})
		}
		if image.UnprocessedImageRef.DigestRef == o.DigestRef() {
			bundleProcessedImage = image
		}
	}

	if len(locationsCfg.Images) != len(o.imagesRef) {
		panic(fmt.Sprintf("Expected: %d images to be written to Location OCI. Actual: %d were written", len(o.imagesRef), len(locationsCfg.Images)))
	}

	destinationRef, err := regname.NewDigest(bundleProcessedImage.DigestRef)
	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency: '%s' have to be a digest", bundleProcessedImage.DigestRef))
	}

	logger.Debugf("creating Locations OCI Image\n")
	// Using NewNoopUI because we do not want to have output from this push
	err = NewLocations(logger).Save(reg, destinationRef, locationsCfg, goui.NewNoopUI())
	if err != nil {
		return err
	}
	return nil
}

func (o *Bundle) Pull(outputPath string, ui goui.UI, pullNestedBundles bool) error {
	isRootBundleRelocated, err := o.pull(outputPath, ui, pullNestedBundles, "", map[string]bool{}, 0)
	if err != nil {
		return err
	}

	ui.BeginLinef("\nLocating image lock file images...\n")
	if isRootBundleRelocated {
		ui.BeginLinef("The bundle repo (%s) is hosting every image specified in the bundle's Images Lock file (.imgpkg/images.yml)\n", o.Repo())
	} else {
		ui.BeginLinef("One or more images not found in bundle repo; skipping lock file update\n")
	}
	return nil
}

func (o *Bundle) pull(baseOutputPath string, ui goui.UI, pullNestedBundles bool, bundlePath string, imagesProcessed map[string]bool, numSubBundles int) (bool, error) {
	img, err := o.checkedImage()
	if err != nil {
		return false, err
	}

	if o.rootBundle(bundlePath) {
		ui.BeginLinef("Pulling bundle '%s'\n", o.DigestRef())
	} else {
		ui.BeginLinef("Pulling nested bundle '%s'\n", o.DigestRef())
	}

	bundleDigestRef, err := regname.NewDigest(o.plainImg.DigestRef())
	if err != nil {
		return false, err
	}

	loggerBuilder := util.NewLogger(uiBlockWriter{ui})
	err = ctlimg.NewDirImage(filepath.Join(baseOutputPath, bundlePath), img, goui.NewIndentingUI(ui)).AsDirectory()
	if err != nil {
		return false, fmt.Errorf("Extracting bundle into directory: %s", err)
	}

	imagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(baseOutputPath, bundlePath, ImgpkgDir, ImagesLockFile))
	if err != nil {
		return false, err
	}

	bundleImageRefs, err := NewImageRefsFromImagesLock(imagesLock, LocationsConfig{
		logger:          loggerBuilder.NewLevelLogger(util.LogWarn, loggerBuilder.NewPrefixedWriter("")),
		imgRetriever:    o.imgRetriever,
		bundleDigestRef: bundleDigestRef,
	})
	if err != nil {
		return false, err
	}

	isRelocatedToBundle, err := bundleImageRefs.UpdateRelativeToRepo(o.imgRetriever, o.Repo())
	if err != nil {
		return false, err
	}

	if pullNestedBundles {
		for _, bundleImgRef := range bundleImageRefs.ImageRefs() {
			if isBundle, alreadyProcessedImage := imagesProcessed[bundleImgRef.Image]; alreadyProcessedImage {
				if isBundle {
					goui.NewIndentingUI(ui).BeginLinef("Pulling nested bundle '%s'\n", bundleImgRef.Image)
					goui.NewIndentingUI(ui).BeginLinef("Skipped, already downloaded\n")
				}
				continue
			}

			subBundle := NewBundle(bundleImgRef.PrimaryLocation(), o.imgRetriever)

			var isBundle bool
			if bundleImgRef.IsBundle != nil {
				isBundle = *bundleImgRef.IsBundle
			} else {
				isBundle, err = subBundle.IsBundle()
				if err != nil {
					return false, err
				}
			}

			imagesProcessed[bundleImgRef.Image] = isBundle

			if !isBundle {
				continue
			}

			numSubBundles++

			if o.shouldPrintNestedBundlesHeader(bundlePath, numSubBundles) {
				ui.BeginLinef("\nNested bundles\n")
			}
			bundleDigest, err := regname.NewDigest(bundleImgRef.Image)
			if err != nil {
				return false, err
			}
			_, err = subBundle.pull(baseOutputPath, goui.NewIndentingUI(ui), pullNestedBundles, o.subBundlePath(bundleDigest), imagesProcessed, numSubBundles)
			if err != nil {
				return false, err
			}
		}
	}

	if isRelocatedToBundle {
		err := bundleImageRefs.ImagesLock().WriteToPath(filepath.Join(baseOutputPath, bundlePath, ImgpkgDir, ImagesLockFile))
		if err != nil {
			return false, fmt.Errorf("Rewriting image lock file: %s", err)
		}
	}

	return isRelocatedToBundle, nil
}

func (*Bundle) subBundlePath(bundleDigest regname.Digest) string {
	return filepath.Join(ImgpkgDir, BundlesDir, strings.ReplaceAll(bundleDigest.DigestStr(), "sha256:", "sha256-"))
}

func (o *Bundle) shouldPrintNestedBundlesHeader(bundlePath string, bundlesProcessed int) bool {
	return o.rootBundle(bundlePath) && bundlesProcessed == 1
}

func (o *Bundle) rootBundle(bundlePath string) bool {
	return bundlePath == ""
}

func (o *Bundle) checkedImage() (regv1.Image, error) {
	isBundle, err := o.IsBundle()
	if err != nil {
		return nil, fmt.Errorf("Checking if image is bundle: %s", err)
	}
	if !isBundle {
		return nil, notABundleError{}
	}

	img, err := o.plainImg.Fetch()
	if err == nil && img == nil {
		panic("Unreachable")
	}
	return img, err
}

type uiBlockWriter struct {
	ui goui.UI
}

var _ io.Writer = uiBlockWriter{}

func (w uiBlockWriter) Write(p []byte) (n int, err error) {
	w.ui.PrintBlock(p)
	return len(p), nil
}
