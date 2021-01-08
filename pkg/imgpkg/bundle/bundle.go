// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	plainimg "github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
)

const (
	BundleConfigLabel = "dev.carvel.imgpkg.bundle"
)

type Bundle struct {
	plainImg *plainimg.PlainImage
	registry ctlimg.Registry
}

func NewBundle(ref string, registry ctlimg.Registry) *Bundle {
	return &Bundle{plainimg.NewPlainImage(ref, registry), registry}
}

func NewBundleFromPlainImage(plainImg *plainimg.PlainImage, registry ctlimg.Registry) *Bundle {
	return &Bundle{plainImg, registry}
}

func (o *Bundle) DigestRef() string { return o.plainImg.DigestRef() }
func (o *Bundle) Tag() string       { return o.plainImg.Tag() }

func (o *Bundle) IsBundle() (bool, error) {
	img, err := o.plainImg.Fetch()
	if err != nil {
		return false, err
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return false, err
	}
	_, present := cfg.Config.Labels[BundleConfigLabel]
	return present, nil
}

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

	err = o.rewriteImagesLockFile(outputPath, ui)
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
		// TODO wrong abstraction level for err msg hint
		return nil, fmt.Errorf("Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
	}
	return o.plainImg.Fetch()
}

func (o *Bundle) rewriteImagesLockFile(outputPath string, ui ui.UI) error {
	path := filepath.Join(outputPath, ImgpkgDir, ImagesLockFile)

	imagesLock, err := lockconfig.NewImagesLockFromPath(path)
	if err != nil {
		return err
	}

	ui.BeginLinef("Locating image lock file images...\n")

	skipped, err := o.localizeImagesLock(&imagesLock)
	if err != nil {
		return err
	}
	if skipped {
		ui.BeginLinef("One or more images not found in bundle repo; skipping lock file update\n")
		return nil
	}

	ui.BeginLinef("All images found in bundle repo; updating lock file: %s\n", path)

	return imagesLock.WriteToPath(path)
}

func (o *Bundle) localizeImagesLock(imagesLock *lockconfig.ImagesLock) (bool, error) {
	var imageRefs []lockconfig.ImageRef

	for _, imgRef := range imagesLock.Images {
		imageInBundleRepo, err := o.imageRelativeToBundle(imgRef.Image)
		if err != nil {
			return false, err
		}

		foundImg, err := o.checkImagesExist([]string{imageInBundleRepo, imgRef.Image})
		if err != nil {
			return false, err
		}

		// If cannot find the image in the bundle repo, will not localize any image
		// We assume that the bundle was not copied to the bundle location,
		// so there we cannot localize any image
		if foundImg != imageInBundleRepo {
			return true, nil
		}

		imageRefs = append(imageRefs, lockconfig.ImageRef{
			Image:       foundImg,
			Annotations: imgRef.Annotations,
		})
	}

	imagesLock.Images = imageRefs
	return false, nil
}

func (o *Bundle) checkImagesExist(urls []string) (string, error) {
	var err error
	for _, img := range urls {
		ref, parseErr := regname.NewDigest(img)
		if parseErr != nil {
			return "", parseErr
		}
		_, err = o.registry.Generic(ref)
		if err == nil {
			return img, nil
		}
	}
	return "", fmt.Errorf("Checking image existance: %s", err)
}

func (o *Bundle) imageRelativeToBundle(img string) (string, error) {
	imgParts := strings.Split(img, "@")
	if len(imgParts) != 2 {
		return "", fmt.Errorf("Parsing image URL: %s", img)
	}
	return o.plainImg.Repo() + "@" + imgParts[1], nil
}
