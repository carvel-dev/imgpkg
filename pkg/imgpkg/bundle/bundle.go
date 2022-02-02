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
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	ctlimg "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/image"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	plainimg "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/plainimage"
)

const (
	BundleConfigLabel = "dev.carvel.imgpkg.bundle"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesLockReader
type ImagesLockReader interface {
	Read(img regv1.Image) (lockconfig.ImagesLock, error)
}

type ImagesMetadata interface {
	Get(regname.Reference) (*regremote.Descriptor, error)
	Image(regname.Reference) (regv1.Image, error)
	Digest(regname.Reference) (regv1.Hash, error)
	FirstImageExists(digests []string) (string, error)
}

type Bundle struct {
	plainImg         *plainimg.PlainImage
	imgRetriever     ImagesMetadata
	imagesLockReader ImagesLockReader

	// cachedImageRefs stores set of ImageRefs that were
	// discovered as part of reading the bundle.
	// Includes refs only directly referenced by the bundle.
	cachedImageRefs map[string]ImageRef
}

func NewBundle(ref string, imagesMetadata ImagesMetadata) *Bundle {
	return NewBundleWithReader(ref, imagesMetadata, &singleLayerReader{})
}

func NewBundleFromPlainImage(plainImg *plainimg.PlainImage, imagesMetadata ImagesMetadata) *Bundle {
	return &Bundle{plainImg: plainImg, imgRetriever: imagesMetadata,
		imagesLockReader: &singleLayerReader{}}
}

func NewBundleWithReader(ref string, imagesMetadata ImagesMetadata, imagesLockReader ImagesLockReader) *Bundle {
	return &Bundle{plainImg: plainimg.NewPlainImage(ref, imagesMetadata),
		imgRetriever: imagesMetadata, imagesLockReader: imagesLockReader}
}

func (o *Bundle) DigestRef() string { return o.plainImg.DigestRef() }
func (o *Bundle) Repo() string      { return o.plainImg.Repo() }
func (o *Bundle) Tag() string       { return o.plainImg.Tag() }

func (o *Bundle) updateCachedImageRef(ref ImageRef) {
	o.cachedImageRefs[ref.Image] = ref.DeepCopy()
}

func (o *Bundle) findCachedImageRef(digestRef string) (ImageRef, bool) {
	ref, found := o.cachedImageRefs[digestRef]
	if found {
		return ref.DeepCopy(), true
	}

	for _, imgRef := range o.cachedImageRefs {
		for _, loc := range imgRef.Locations() {
			if loc == digestRef {
				return imgRef.DeepCopy(), true
			}
		}
	}

	return ImageRef{}, false
}

func (o *Bundle) allCachedImageRefs() []ImageRef {
	var imgsRef []ImageRef
	for _, ref := range o.cachedImageRefs {
		imgsRef = append(imgsRef, ref.DeepCopy())
	}
	return imgsRef
}

// NoteCopy writes an image-location representing the bundle / images that have been copied
func (o *Bundle) NoteCopy(processedImages *imageset.ProcessedImages, reg ImagesMetadataWriter, ui util.UIWithLevels) error {
	locationsCfg := ImageLocationsConfig{
		APIVersion: LocationAPIVersion,
		Kind:       ImageLocationsKind,
	}
	var bundleProcessedImage imageset.ProcessedImage
	for _, image := range processedImages.All() {
		ref, found := o.findCachedImageRef(image.UnprocessedImageRef.DigestRef)
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

	if len(locationsCfg.Images) != len(o.cachedImageRefs) {
		panic(fmt.Sprintf("Expected: %d images to be written to Location OCI. Actual: %d were written", len(o.cachedImageRefs), len(locationsCfg.Images)))
	}

	destinationRef, err := regname.NewDigest(bundleProcessedImage.DigestRef)
	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency: '%s' have to be a digest", bundleProcessedImage.DigestRef))
	}

	ui.Debugf("creating Locations OCI Image\n")

	// Using NewNoopUI because we do not want to have output from this push
	return NewLocations(ui).Save(reg, destinationRef, locationsCfg, goui.NewNoopUI())
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

	err = ctlimg.NewDirImage(filepath.Join(baseOutputPath, bundlePath), img, goui.NewIndentingUI(ui)).AsDirectory()
	if err != nil {
		return false, fmt.Errorf("Extracting bundle into directory: %s", err)
	}

	imagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(baseOutputPath, bundlePath, ImgpkgDir, ImagesLockFile))
	if err != nil {
		return false, err
	}

	bundleImageRefs, err := NewImageRefsFromImagesLock(imagesLock, LocationsConfig{
		ui:              util.NewUILevelLogger(util.LogWarn, ui),
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
