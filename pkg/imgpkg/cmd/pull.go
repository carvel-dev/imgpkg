// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"

	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"carvel.dev/imgpkg/pkg/imgpkg/lockconfig"
	v1 "carvel.dev/imgpkg/pkg/imgpkg/v1"
	"github.com/cppforlife/go-cli-ui/ui"
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

	levelLogger := util.NewUILevelLogger(util.LogWarn, util.NewLogger(po.ui))
	imageRef := ""
	switch {
	case len(po.LockInputFlags.LockFilePath) > 0:
		if len(po.LockInputFlags.LockFilePath) > 0 {
			bundleLock, err := lockconfig.NewBundleLockFromPath(po.LockInputFlags.LockFilePath)
			if err != nil {
				return err
			}
			imageRef = bundleLock.Bundle.Image
		}
	case len(po.BundleFlags.Bundle) > 0:
		imageRef = po.BundleFlags.Bundle
	case len(po.ImageFlags.Image) > 0:
		imageRef = po.ImageFlags.Image
	default:
		panic("Unreachable code")
	}

	pullOpts := v1.PullOpts{
		Logger:   levelLogger,
		AsImage:  !po.ImageIsBundleCheck,
		IsBundle: len(po.ImageFlags.Image) == 0,
	}
	if po.BundleRecursiveFlags.Recursive {
		_, err = v1.PullRecursive(imageRef, po.OutputPath, pullOpts, po.RegistryFlags.AsRegistryOpts())
	} else {
		_, err = v1.Pull(imageRef, po.OutputPath, pullOpts, po.RegistryFlags.AsRegistryOpts())
	}

	if errors.Is(err, &v1.ErrIsBundle{}) {
		if len(po.ImageFlags.Image) == 0 {
			if po.ImageIsBundleCheck {
				return fmt.Errorf("Expected bundle flag when pulling a bundle (hint: Use -b instead of -i for bundles)")
			}
		} else {
			return fmt.Errorf("Expected bundle flag when pulling a bundle (hint: Use -b instead of -i for bundles)")
		}
	} else if len(po.ImageFlags.Image) == 0 && errors.Is(err, &v1.ErrIsNotBundle{}) {
		return fmt.Errorf("Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
	}

	return err
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

	if po.BundleRecursiveFlags.Recursive && len(po.ImageFlags.Image) > 0 {
		return fmt.Errorf("Cannot use --recursive (-r) flag when pulling a bundle")
	}

	if !po.ImageIsBundleCheck && len(po.BundleFlags.Bundle) != 0 {
		return fmt.Errorf("Cannot set --image-is-bundle-check while using -b flag")
	}
	return nil
}
