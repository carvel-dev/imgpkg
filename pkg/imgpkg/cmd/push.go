// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagelayers"
	"os"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/pkg/imgpkg/plainimage"
	"github.com/spf13/cobra"
)

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
	logger := ctlimg.NewLogger(os.Stderr)
	prefixedLogger := logger.NewPrefixedWriter("push | ")

	registry, err := ctlimg.NewRegistry(o.RegistryFlags.AsRegistryOpts(), imagelayers.ImageLayerWriterChecker{}, prefixedLogger)
	if err != nil {
		return fmt.Errorf("Unable to create a registry with provided options: %v", err)
	}

	var imageURL string

	isBundle := o.BundleFlags.Bundle != ""
	isImage := o.ImageFlags.Image != ""

	switch {
	case isBundle && isImage:
		return fmt.Errorf("Expected only one of image or bundle")

	case !isBundle && !isImage:
		return fmt.Errorf("Expected either image or bundle")

	case isBundle:
		imageURL, err = o.pushBundle(registry)
		if err != nil {
			return err
		}

	case isImage:
		imageURL, err = o.pushImage(registry)
		if err != nil {
			return err
		}

	default:
		panic("Unreachable code")
	}

	o.ui.BeginLinef("Pushed '%s'", imageURL)

	return nil
}

func (o *PushOptions) pushBundle(registry ctlimg.Registry) (string, error) {
	uploadRef, err := regname.NewTag(o.BundleFlags.Bundle, regname.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("Parsing '%s': %s", o.BundleFlags.Bundle, err)
	}

	imageURL, err := bundle.NewContents(o.FileFlags.Files, o.FileFlags.ExcludedFilePaths).Push(uploadRef, registry, o.ui)
	if err != nil {
		return "", err
	}

	if o.LockOutputFlags.LockFilePath != "" {
		bundleLock := lockconfig.BundleLock{
			LockVersion: lockconfig.LockVersion{
				APIVersion: lockconfig.BundleLockAPIVersion,
				Kind:       lockconfig.BundleLockKind,
			},
			Bundle: lockconfig.BundleRef{
				Image: imageURL,
				Tag:   uploadRef.TagStr(),
			},
		}

		err := bundleLock.WriteToPath(o.LockOutputFlags.LockFilePath)
		if err != nil {
			return "", err
		}
	}

	return imageURL, nil
}

func (o *PushOptions) pushImage(registry ctlimg.Registry) (string, error) {
	if o.LockOutputFlags.LockFilePath != "" {
		return "", fmt.Errorf("Lock output is not compatible with image, use bundle for lock output")
	}

	uploadRef, err := regname.NewTag(o.ImageFlags.Image, regname.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("Parsing '%s': %s", o.ImageFlags.Image, err)
	}

	isBundle, err := bundle.NewContents(o.FileFlags.Files, o.FileFlags.ExcludedFilePaths).PresentsAsBundle()
	if err != nil {
		return "", err
	}
	if isBundle {
		return "", fmt.Errorf("Images cannot be pushed with '.imgpkg' directories, consider using --bundle (-b) option")
	}

	return plainimage.NewContents(o.FileFlags.Files, o.FileFlags.ExcludedFilePaths).Push(uploadRef, nil, registry, o.ui)
}
