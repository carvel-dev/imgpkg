// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"path/filepath"
	"strings"

	goui "github.com/cppforlife/go-cli-ui/ui"
	"github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	plainimg "github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
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
}

func NewBundle(ref string, imagesMetadata ctlimg.ImagesMetadata) *Bundle {
	return NewBundleWithReader(ref, imagesMetadata, &singleLayerReader{})
}

func NewBundleFromPlainImage(plainImg *plainimg.PlainImage, imagesMetadata ctlimg.ImagesMetadata) *Bundle {
	return &Bundle{plainImg, imagesMetadata, &singleLayerReader{}}
}

func NewBundleWithReader(ref string, imagesMetadata ctlimg.ImagesMetadata, imagesLockReader ImagesLockReader) *Bundle {
	return &Bundle{plainimg.NewPlainImage(ref, imagesMetadata), imagesMetadata, imagesLockReader}
}

func (o *Bundle) DigestRef() string { return o.plainImg.DigestRef() }
func (o *Bundle) Repo() string      { return o.plainImg.Repo() }
func (o *Bundle) Tag() string       { return o.plainImg.Tag() }

func (o *Bundle) Pull(outputPath string, ui goui.UI, pullNestedBundles bool) error {
	return o.pull(outputPath, ui, pullNestedBundles, "", map[string]bool{}, 0)
}

func (o *Bundle) pull(baseOutputPath string, ui goui.UI, pullNestedBundles bool, bundlePath string, imagesProcessed map[string]bool, numSubBundles int) error {
	img, err := o.checkedImage()
	if err != nil {
		return err
	}

	if o.rootBundle(bundlePath) {
		ui.BeginLinef("Pulling bundle '%s'\n", o.DigestRef())
		ui.BeginLinef("Bundle Layers\n")
	} else {
		ui.BeginLinef("Pulling nested bundle '%s'\n", o.DigestRef())
	}

	err = ctlimg.NewDirImage(filepath.Join(baseOutputPath, bundlePath), img, ui).AsDirectory()
	if err != nil {
		return fmt.Errorf("Extracting bundle into directory: %s", err)
	}

	imagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(baseOutputPath, bundlePath, ImgpkgDir, ImagesLockFile))
	if err != nil {
		return err
	}

	if pullNestedBundles {
		for _, image := range imagesLock.Images {
			if isBundle, alreadyProcessedImage := imagesProcessed[image.Image]; alreadyProcessedImage {
				if isBundle {
					goui.NewIndentingUI(ui).BeginLinef("Pulling nested bundle '%s'\n", image.Image)
					goui.NewIndentingUI(ui).BeginLinef("Skipped, already downloaded\n")
				}
				continue
			}

			subBundle := NewBundle(image.Image, o.imgRetriever)
			isBundle, err := subBundle.IsBundle()
			if err != nil {
				return err
			}
			imagesProcessed[image.Image] = isBundle

			if !isBundle {
				continue
			}

			numSubBundles++

			if o.shouldPrintNestedBundlesHeader(bundlePath, numSubBundles) {
				ui.BeginLinef("\nNested bundles\n")
			}
			bundleDigest, err := name.NewDigest(image.Image)
			if err != nil {
				return err
			}
			err = subBundle.pull(baseOutputPath, goui.NewIndentingUI(ui), pullNestedBundles, subBundlePath(bundleDigest), imagesProcessed, numSubBundles)
			if err != nil {
				return err
			}
		}
	}

	imagesLockUI := ui
	if !o.rootBundle(bundlePath) {
		imagesLockUI = goui.NewWriterUI(noopWriter{}, noopWriter{}, goui.NoopLogger{})
	}

	err = NewImagesLock(imagesLock, o.imgRetriever, o.Repo()).WriteToPath(filepath.Join(baseOutputPath, bundlePath), imagesLockUI)
	if err != nil {
		return fmt.Errorf("Rewriting image lock file: %s", err)
	}

	return nil
}

func subBundlePath(bundleDigest name.Digest) string {
	return filepath.Join(ImgpkgDir, SubBundlesDir, strings.ReplaceAll(bundleDigest.DigestStr(), "sha256:", "sha256-"))
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

type noopWriter struct{}

func (noopWriter) Write(_ []byte) (n int, err error) {
	return 0, nil
}
