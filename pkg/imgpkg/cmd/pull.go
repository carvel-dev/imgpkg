// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
	"github.com/k14s/imgpkg/pkg/imgpkg/registry"
	"github.com/spf13/cobra"
)

type PullOptions struct {
	ui ui.UI

	ImageFlags           ImageFlags
	ImageIsBundleCheck   bool
	RegistryFlags        RegistryFlags
	BundleFlags          BundleFlags
	LockInputFlags       LockInputFlags
	BundleRecursiveFlags BundleRecursiveFlags
	OutputPath           string
}

func NewPullOptions(ui ui.UI) *PullOptions {
	return &PullOptions{ui: ui}
}

func NewPullCmd(o *PullOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull files from bundle, image, or bundle lock file",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
		Example: `
  # Pull bundle repo/app1-bundle and extract into /tmp/app1-bundle
  imgpkg pull -b repo/app1-bundle -o /tmp/app1-bundle

  # Pull image repo/app1-image and extract into /tmp/app1-image
  imgpkg pull -i repo/app1-image -o /tmp/app1-image`,
	}
	o.ImageFlags.Set(cmd)
	cmd.Flags().BoolVar(&o.ImageIsBundleCheck, "image-is-bundle-check", true, "Error when image is a bundle (disable pulling bundles via -i)")
	o.RegistryFlags.Set(cmd)
	o.BundleFlags.Set(cmd)
	o.BundleRecursiveFlags.Set(cmd)
	o.LockInputFlags.Set(cmd)
	cmd.Flags().StringVarP(&o.OutputPath, "output", "o", "", "Output directory path")
	cmd.MarkFlagRequired("output")

	return cmd
}

func (po *PullOptions) Run() error {
	err := po.validate()
	if err != nil {
		return err
	}

	reg, err := registry.NewRegistry(po.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return err
	}

	switch {
	case len(po.LockInputFlags.LockFilePath) > 0 || len(po.BundleFlags.Bundle) > 0:
		bundleRef := po.BundleFlags.Bundle

		if len(po.LockInputFlags.LockFilePath) > 0 {
			bundleLock, err := lockconfig.NewBundleLockFromPath(po.LockInputFlags.LockFilePath)
			if err != nil {
				return err
			}
			bundleRef = bundleLock.Bundle.Image
		}

		err := bundle.NewBundle(bundleRef, reg).Pull(po.OutputPath, po.ui, po.BundleRecursiveFlags.Recursive)
		if err != nil {
			if bundle.IsNotBundleError(err) {
				return fmt.Errorf("Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
			}
			return err
		}
		return nil

	case len(po.ImageFlags.Image) > 0:
		plainImg := plainimage.NewPlainImage(po.ImageFlags.Image, reg)

		isImage, err := plainImg.IsImage()
		if err != nil {
			return err
		}
		if !isImage {
			return fmt.Errorf("Unable to pull non-images, such as image indexes. (hint: provide a specific digest to the image instead)")
		}

		if po.ImageIsBundleCheck {
			isBundle, err := bundle.NewBundleFromPlainImage(plainImg, reg).IsBundle()
			if err != nil {
				return err
			}
			if isBundle {
				return fmt.Errorf("Expected bundle flag when pulling a bundle (hint: Use -b instead of -i for bundles)")
			}
		}

		return plainImg.Pull(po.OutputPath, po.ui)

	default:
		panic("Unreachable code")
	}
}

func (po *PullOptions) validate() error {
	if po.OutputPath == "" {
		return fmt.Errorf("Expected --output to be none empty")
	}

	if po.OutputPath == "/" || po.OutputPath == "." || po.OutputPath == ".." {
		return fmt.Errorf("Disallowed output directory (trying to avoid accidental deletion)")
	}

	presentInputParams := 0
	for _, inputParam := range []string{po.LockInputFlags.LockFilePath, po.BundleFlags.Bundle, po.ImageFlags.Image} {
		if len(inputParam) > 0 {
			presentInputParams++
		}
	}
	if presentInputParams > 1 {
		return fmt.Errorf("Expected only one of image, bundle, or lock")
	}
	if presentInputParams == 0 {
		return fmt.Errorf("Expected either image or bundle reference")
	}
	return nil
}
