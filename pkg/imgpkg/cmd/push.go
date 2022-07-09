// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/plainimage"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
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
  # Push bundle repo/app1-config with contents of config/ directory
  imgpkg push -b repo/app1-config -f config/

  # Push image repo/app1-config with contents from multiple locations
  imgpkg push -i repo/app1-config -f config/ -f additional-config.yml`,
	}
	o.ImageFlags.Set(cmd)
	o.BundleFlags.Set(cmd)
	o.LockOutputFlags.SetOnPush(cmd)
	o.FileFlags.Set(cmd)
	o.RegistryFlags.Set(cmd)
	return cmd
}

func (po *PushOptions) Run() error {
	reg, err := registry.NewSimpleRegistry(po.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return err
	}

	var imageURL string

	isBundle := po.BundleFlags.Bundle != ""
	isImage := po.ImageFlags.Image != ""

	switch {
	case isBundle && isImage:
		return fmt.Errorf("Expected only one of image or bundle")

	case !isBundle && !isImage:
		return fmt.Errorf("Expected either image or bundle")

	case isBundle:
		imageURL, err = po.pushBundle(reg)
		if err != nil {
			return err
		}

	case isImage:
		imageURL, err = po.pushImage(reg)
		if err != nil {
			return err
		}

	default:
		panic("Unreachable code")
	}

	po.ui.BeginLinef("Pushed '%s'", imageURL)

	return nil
}

func (po *PushOptions) pushBundle(registry registry.Registry) (string, error) {
	uploadRef, err := regname.NewTag(po.BundleFlags.Bundle, regname.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("Parsing '%s': %s", po.BundleFlags.Bundle, err)
	}

	imageURL, err := bundle.NewContents(po.FileFlags.Files, po.FileFlags.ExcludedFilePaths).Push(uploadRef, registry, po.ui)
	if err != nil {
		return "", err
	}

	if po.LockOutputFlags.LockFilePath != "" {
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

		err := bundleLock.WriteToPath(po.LockOutputFlags.LockFilePath)
		if err != nil {
			return "", err
		}
	}

	return imageURL, nil
}

func (po *PushOptions) pushImage(registry registry.Registry) (string, error) {
	if po.LockOutputFlags.LockFilePath != "" {
		return "", fmt.Errorf("Lock output is not compatible with image, use bundle for lock output")
	}

	uploadRef, err := regname.NewTag(po.ImageFlags.Image, regname.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("Parsing '%s': %s", po.ImageFlags.Image, err)
	}

	isBundle, err := bundle.NewContents(po.FileFlags.Files, po.FileFlags.ExcludedFilePaths).PresentsAsBundle()
	if err != nil {
		return "", err
	}
	if isBundle {
		return "", fmt.Errorf("Images cannot be pushed with '.imgpkg' directories, consider using --bundle (-b) option")
	}
	fmt.Printf("labels: %v", po.ImageFlags.Labels)

	return plainimage.NewContents(po.FileFlags.Files, po.FileFlags.ExcludedFilePaths).Push(uploadRef, po.ImageFlags.Labels, registry, po.ui)
}
