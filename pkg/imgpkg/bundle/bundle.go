// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"path/filepath"

	"github.com/cppforlife/go-cli-ui/ui"
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

func (o *Bundle) Pull(outputPath string, ui ui.UI) error {
	return o.pull(outputPath, "", map[string]interface{}{}, ui)
}

func (o *Bundle) pull(baseOutputPath string, bundlePath string, bundlesProcessed map[string]interface{}, ui ui.UI) error {
	img, err := o.checkedImage()
	if err != nil {
		return err
	}

	ui.BeginLinef("Pulling bundle '%s'\n", o.DigestRef())

	err = ctlimg.NewDirImage(filepath.Join(baseOutputPath, bundlePath), img, ui).AsDirectory()
	if err != nil {
		return fmt.Errorf("Extracting bundle into directory: %s", err)
	}

	imagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(baseOutputPath, bundlePath, ImgpkgDir, ImagesLockFile))
	if err != nil {
		return err
	}

	for _, image := range imagesLock.Images {
		//TODO: run in a go routine?
		//TODO: handle cyclic bundle references

		subBundle := NewBundle(image.Image, o.imgRetriever)
		isBundle, err := subBundle.IsBundle()
		if err != nil {
			return err
		}
		if _, alreadyProcessedBundle := bundlesProcessed[image.Image]; alreadyProcessedBundle || !isBundle {
			continue
		}
		bundlesProcessed[image.Image] = nil

		reference, err := name.NewDigest(image.Image)
		if err != nil {
			return err
		}
		err = subBundle.pull(baseOutputPath, filepath.Join(".imgpkg", "bundles", reference.DigestStr()), bundlesProcessed, ui)
		if err != nil {
			return err
		}
	}

	err = NewImagesLock(imagesLock, o.imgRetriever, o.Repo()).WriteToPath(filepath.Join(baseOutputPath, bundlePath), ui)
	if err != nil {
		return fmt.Errorf("Rewriting image lock file: %s", err)
	}

	return nil
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
