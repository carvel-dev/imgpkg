// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package plainimage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
)

type Contents struct {
	paths         []string
	excludedPaths []string
}

type ImagesWriter interface {
	WriteImage(regname.Reference, regv1.Image) error
}

func NewContents(paths []string, excludedPaths []string) Contents {
	return Contents{paths: paths, excludedPaths: excludedPaths}
}

func (i Contents) Push(uploadRef regname.Tag, labels map[string]string, writer ImagesWriter, ui ui.UI) (string, error) {
	err := i.validate()
	if err != nil {
		return "", err
	}

	tarImg := ctlimg.NewTarImage(i.paths, i.excludedPaths, InfoLog{ui})

	img, err := tarImg.AsFileImage(labels)
	if err != nil {
		return "", err
	}

	defer img.Remove()

	err = writer.WriteImage(uploadRef, img)
	if err != nil {
		return "", fmt.Errorf("Writing '%s': %s", uploadRef.Name(), err)
	}

	digest, err := img.Digest()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s@%s", uploadRef.Context(), digest), nil
}

func (i Contents) validate() error {
	return i.checkRepeatedPaths()
}

func (i Contents) checkRepeatedPaths() error {
	imageRootPaths := make(map[string][]string)
	for _, flagPath := range i.paths {
		err := filepath.Walk(flagPath, func(currPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			imageRootPath, err := filepath.Rel(flagPath, currPath)
			if err != nil {
				return err
			}

			if imageRootPath == "." {
				if info.IsDir() {
					return nil
				}
				imageRootPath = filepath.Base(flagPath)
			}
			imageRootPaths[imageRootPath] = append(imageRootPaths[imageRootPath], currPath)
			return nil
		})

		if err != nil {
			return err
		}
	}

	var repeatedPaths []string
	for _, v := range imageRootPaths {
		if len(v) > 1 {
			repeatedPaths = append(repeatedPaths, v...)
		}
	}
	if len(repeatedPaths) > 0 {
		return fmt.Errorf("Found duplicate paths: %s", strings.Join(repeatedPaths, ", "))
	}
	return nil
}
