// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// TODO rename when we have a name
const BundleDir = ".imgpkg"
const ImageLockFile = "images.yml"

type PushOptions struct {
	ui ui.UI

	ImageFlags      ImageFlags
	BundleFlags     BundleFlags
	LockOutputFlags LockOutputFlags
	FileFlags       FileFlags
	RegistryFlags   RegistryFlags
}

func NewPushOptions(ui ui.UI) *PushOptions {
	return &PushOptions{ui: ui}
}

func NewPushCmd(o *PushOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push files as image",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
		Example: `
  # Push bundle dkalinin/app1-config with contents of config/ directory
  imgpkg push -b dkalinin/app1-config -f config/

  # Push image dkalinin/app1-config with contents from multiple locations
  imgpkg push -i dkalinin/app1-config -f config/ -f additional-config.yml`,
	}
	o.ImageFlags.Set(cmd)
	o.BundleFlags.Set(cmd)
	o.LockOutputFlags.Set(cmd)
	o.FileFlags.Set(cmd)
	o.RegistryFlags.Set(cmd)
	return cmd
}

func (o *PushOptions) Run() error {
	var inputRef string
	var registry ctlimg.Registry
	var err error

	switch {
	case o.isBundle() && o.isImage():
		return fmt.Errorf("Expected only one of image or bundle")

	case !o.isBundle() && !o.isImage():
		return fmt.Errorf("Expected either image or bundle")

	case o.isBundle():
		registry, err = ctlimg.NewRegistry(o.RegistryFlags.AsRegistryOpts())
		if err != nil {
			return fmt.Errorf("Unable to create a registry with the options %v: %v", o.RegistryFlags.AsRegistryOpts(), err)
		}
		err = o.validateBundle(registry)
		if err != nil {
			return err
		}

		inputRef = o.BundleFlags.Bundle

	case o.isImage():
		if o.LockOutputFlags.LockFilePath != "" {
			return fmt.Errorf("Lock output is not compatible with image, use bundle for lock output")
		}

		bundleDirPaths, err := o.findBundleDirs()
		if err != nil {
			return err
		}

		if len(bundleDirPaths) > 0 {
			return fmt.Errorf("Images cannot be pushed with '%s' directories (found %d at '%s'), consider using a bundle", BundleDir, len(bundleDirPaths), strings.Join(bundleDirPaths, ","))
		}
		registry, err = ctlimg.NewRegistry(o.RegistryFlags.AsRegistryOpts())
		if err != nil {
			return fmt.Errorf("Unable to create a registry with the options %v: %v", o.RegistryFlags.AsRegistryOpts(), err)
		}
		inputRef = o.ImageFlags.Image
	}

	err = o.checkRepeatedPaths()
	if err != nil {
		return err
	}

	uploadRef, err := regname.NewTag(inputRef, regname.WeakValidation)
	if err != nil {
		return fmt.Errorf("Parsing '%s': %s", inputRef, err)
	}

	var img *ctlimg.FileImage
	tarImg := ctlimg.NewTarImage(o.FileFlags.Files, o.FileFlags.FileExcludeDefaults, InfoLog{o.ui})
	if o.isBundle() {
		img, err = tarImg.AsFileBundle()
	} else {
		img, err = tarImg.AsFileImage()
	}

	if err != nil {
		return err
	}

	defer img.Remove()

	err = registry.WriteImage(uploadRef, img)
	if err != nil {
		return fmt.Errorf("Writing '%s': %s", uploadRef.Name(), err)
	}

	digest, err := img.Digest()
	if err != nil {
		return err
	}

	imageURL := fmt.Sprintf("%s@%s", uploadRef.Context(), digest)

	o.ui.BeginLinef("Pushed '%s'", imageURL)

	if o.LockOutputFlags.LockFilePath != "" {
		bundleLock := BundleLock{
			ApiVersion: BundleLockAPIVersion,
			Kind:       BundleLockKind,
			Spec: BundleSpec{
				Image: ImageLocation{
					DigestRef:   imageURL,
					OriginalTag: uploadRef.TagStr(),
				},
			},
		}

		manifestBs, err := yaml.Marshal(bundleLock)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(o.LockOutputFlags.LockFilePath, append([]byte("---\n"), manifestBs...), 0700)
		if err != nil {
			return fmt.Errorf("Writing lock file: %s", err)
		}
	}

	return nil
}

func (o *PushOptions) validateBundleDirs(bundleDirPaths []string) error {
	if len(bundleDirPaths) != 1 {
		return fmt.Errorf("Expected one '%s' dir, got %d: %s", BundleDir, len(bundleDirPaths), strings.Join(bundleDirPaths, ", "))
	}

	path := bundleDirPaths[0]

	// make sure it is a child of one input dir
	for _, flagPath := range o.FileFlags.Files {
		flagPath, err := filepath.Abs(flagPath)
		if err != nil {
			return err
		}

		if filepath.Dir(path) == flagPath {
			return nil
		}
	}

	return fmt.Errorf("Expected '%s' directory, to be a direct child of one of: %s; was %s", BundleDir, strings.Join(o.FileFlags.Files, ", "), path)
}

func (o *PushOptions) findBundleDirs() ([]string, error) {
	var bundlePaths []string
	for _, flagPath := range o.FileFlags.Files {
		err := filepath.Walk(flagPath, func(currPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if filepath.Base(currPath) != BundleDir {
				return nil
			}

			currPath, err = filepath.Abs(currPath)
			if err != nil {
				return err
			}

			bundlePaths = append(bundlePaths, currPath)

			return nil
		})

		if err != nil {
			return []string{}, err
		}
	}

	return bundlePaths, nil
}

func (o *PushOptions) validateBundle(registry ctlimg.Registry) error {
	bundlePaths, err := o.findBundleDirs()
	if err != nil {
		return nil
	}

	err = o.validateBundleDirs(bundlePaths)
	if err != nil {
		return err
	}

	imagesBytes, err := ioutil.ReadFile(filepath.Join(bundlePaths[0], ImageLockFile))
	if err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("Must have images.yml in '%s' directory", BundleDir)
		}
		return err
	}

	var imgLock ImageLock
	err = yaml.Unmarshal(imagesBytes, &imgLock)
	if err != nil {
		return fmt.Errorf("Unmarshalling image lock: %s", err)
	}

	bundles, err := imgLock.CheckForBundles(registry)
	if err != nil {
		return fmt.Errorf("Checking image lock for bundles: %s", err)
	}
	if len(bundles) != 0 {
		return fmt.Errorf("Expected image lock to not contain bundle reference: '%v'", strings.Join(bundles, "', '"))
	}
	return nil
}

func (o *PushOptions) checkRepeatedPaths() error {
	imageRootPaths := make(map[string][]string)
	for _, flagPath := range o.FileFlags.Files {
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

func (o *PushOptions) isBundle() bool {
	return o.BundleFlags.Bundle != ""
}

func (o *PushOptions) isImage() bool {
	return o.ImageFlags.Image != ""
}
