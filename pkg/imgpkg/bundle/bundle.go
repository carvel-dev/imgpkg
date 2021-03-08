// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"path/filepath"

	"github.com/cppforlife/go-cli-ui/ui"
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
	img, err := o.checkedImage()
	if err != nil {
		return err
	}

	ui.BeginLinef("Pulling bundle '%s'\n", o.DigestRef())

	err = ctlimg.NewDirImage(outputPath, img, ui).AsDirectory()
	if err != nil {
		return fmt.Errorf("Extracting bundle into directory: %s", err)
	}

	imagesLock, err := lockconfig.NewImagesLockFromPath(filepath.Join(outputPath, ImgpkgDir, ImagesLockFile))
	if err != nil {
		return err
	}

	err = NewImagesLock(imagesLock, o.imgRetriever, o.Repo()).WriteToPath(outputPath, ui)
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
