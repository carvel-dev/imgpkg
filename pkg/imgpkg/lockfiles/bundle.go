// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package lockfiles

import (
	"archive/tar"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ImageRetriever interface {
	Image(ref regname.Reference) (regv1.Image, error)
}

type Bundle struct {
	URL   string
	Tag   string
	Image regv1.Image
}

func IsBundle(img regv1.Image) (bool, error) {
	cfg, err := img.ConfigFile()
	if err != nil {
		return false, err
	}

	_, present := cfg.Config.Labels[image.BundleConfigLabel] // TODO: Move this to BundleConfigLabel to a different package
	return present, nil
}

func GetReferencedImages(bundleRef regname.Reference, reg ImageRetriever) ([]ImageDesc, error) {
	img, err := reg.Image(bundleRef)
	if err != nil {
		return nil, err
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}

	if len(layers) != 1 {
		return nil, fmt.Errorf("Expected bundle to only have a single layer, got %d", len(layers))
	}

	layer := layers[0]

	mediaType, err := layer.MediaType()
	if err != nil {
		return nil, err
	}

	if mediaType != types.DockerLayer {
		return nil, fmt.Errorf("Expected layer to have docker layer media type, was %s", mediaType)
	}

	// here we know layer is .tgz so decompress and read tar headers
	unzippedReader, err := layer.Uncompressed()
	if err != nil {
		return nil, fmt.Errorf("Could not read bundle image layer contents: %v", err)
	}

	tarReader := tar.NewReader(unzippedReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("Expected to find .imgpkg/images.yml in bundle image")
		}

		if err != nil {
			return nil, fmt.Errorf("reading tar: %v", err)
		}

		basename := filepath.Base(header.Name)
		dirname := filepath.Dir(header.Name)
		if dirname == BundleDir && basename == ImageLockFile {
			break
		}
	}

	imgLock := ImageLock{}
	if err := yaml.NewDecoder(tarReader).Decode(&imgLock); err != nil {
		return nil, fmt.Errorf("reading images.yml: %v", err)
	}

	return imgLock.Spec.Images, nil
}

func CollectBundleURLs(bundleLockPath string, reg ImageRetriever) (regname.Reference, string, []regname.Reference, error) {
	bundleLock, err := ReadBundleLockFile(bundleLockPath)
	if err != nil {
		return nil, "", nil, err
	}

	parsedRef, img, err := getRefAndImage(bundleLock.Spec.Image.DigestRef, reg)
	if err != nil {
		return nil, "", nil, err
	}

	ok, err := IsBundle(img)
	if err != nil {
		return nil, "", nil, err
	}
	if !ok {
		return nil, "", nil, fmt.Errorf("expected image flag when given an image reference. Please run with -i instead of -b, or use -b with a bundle reference")
	}

	imgs, err := GetReferencedImages(parsedRef, reg)
	if err != nil {
		return nil, "", nil, err
	}

	var result []regname.Reference
	for _, img := range imgs {
		ref, err := regname.ParseReference(img.Image)
		if err != nil {
			return nil, "", nil, errors.Wrapf(err, fmt.Sprintf("parsing reference for image %s", img.Image))
		}
		result = append(result, ref)
	}

	return parsedRef, bundleLock.Spec.Image.OriginalTag, result, nil
}

func CollectImageLockURLs(imageLockPath string, reg ImageRetriever) ([]regname.Reference, error) {
	imgLock, err := ReadImageLockFile(imageLockPath)
	if err != nil {
		return nil, err
	}

	bundles, err := imgLock.CheckForBundles(reg)
	if err != nil {
		return nil, fmt.Errorf("checking image lock for bundles: %s", err)
	}

	if len(bundles) != 0 {
		return nil, fmt.Errorf("expected not to contain bundle reference: '%v'", strings.Join(bundles, "', '"))
	}

	var result []regname.Reference
	for _, img := range imgLock.Spec.Images {
		ref, err := regname.ParseReference(img.Image)
		if err != nil {
			return nil, errors.Wrapf(err, fmt.Sprintf("parsing reference for image %s", img.Image))
		}
		result = append(result, ref)
	}

	return result, nil
}

func getRefAndImage(ref string, reg ImageRetriever) (regname.Reference, regv1.Image, error) {
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
