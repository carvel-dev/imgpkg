// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagelayers"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
	"github.com/spf13/cobra"
)

type PullOptions struct {
	ui ui.UI

	ImageFlags           ImageFlags
	RegistryFlags        RegistryFlags
	BundleFlags          BundleFlags
	LockInputFlags       LockInputFlags
	BundleRecursiveFlags BundleRecursiveFlags
	OutputPath           string
}

var _ ctlimg.ImagesMetadata = ctlimg.Registry{}

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
	o.RegistryFlags.Set(cmd)
	o.BundleFlags.Set(cmd)
	o.BundleRecursiveFlags.Set(cmd)
	o.LockInputFlags.Set(cmd)
	cmd.Flags().StringVarP(&o.OutputPath, "output", "o", "", "Output directory path")
	cmd.MarkFlagRequired("output")

	return cmd
}

func (o *PullOptions) Run() error {
	err := o.validate()
	if err != nil {
		return err
	}

	registry, err := ctlimg.NewRegistry(o.RegistryFlags.AsRegistryOpts(), imagelayers.ImageLayerWriterFilter{})
	if err != nil {
		return fmt.Errorf("Unable to create a registry with the options %v: %v", o.RegistryFlags.AsRegistryOpts(), err)
	}

	switch {
	case len(o.LockInputFlags.LockFilePath) > 0 || len(o.BundleFlags.Bundle) > 0:
		bundleRef := o.BundleFlags.Bundle

		if len(o.LockInputFlags.LockFilePath) > 0 {
			bundleLock, err := lockconfig.NewBundleLockFromPath(o.LockInputFlags.LockFilePath)
			if err != nil {
				return err
			}
			bundleRef = bundleLock.Bundle.Image
		}

		err := bundle.NewBundle(bundleRef, registry).Pull(o.OutputPath, o.ui, o.BundleRecursiveFlags.Recursive)
		if err != nil {
			if bundle.IsNotBundleError(err) {
				return fmt.Errorf("Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
			}
			return err
		}
		return nil

	case len(o.ImageFlags.Image) > 0:
		plainImg := plainimage.NewPlainImage(o.ImageFlags.Image, registry)
		ok, err := bundle.NewBundleFromPlainImage(plainImg, registry).IsBundle()
		if err != nil {
			return err
		}
		if ok {
			return fmt.Errorf("Expected bundle flag when pulling a bundle (hint: Use -b instead of -i for bundles)")
		}
		return plainImg.Pull(o.OutputPath, o.ui)

	default:
		panic("Unreachable code")
	}
}

func (o *PullOptions) validate() error {
	if o.OutputPath == "" {
		return fmt.Errorf("Expected --output to be none empty")
	}

	if o.OutputPath == "/" || o.OutputPath == "." || o.OutputPath == ".." {
		return fmt.Errorf("Disallowed output directory (trying to avoid accidental deletion)")
	}

	presentInputParams := 0
	for _, inputParam := range []string{o.LockInputFlags.LockFilePath, o.BundleFlags.Bundle, o.ImageFlags.Image} {
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
