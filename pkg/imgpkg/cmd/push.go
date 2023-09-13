// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
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
	LabelFlags      LabelFlags
	TagFlags        TagFlags
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
	o.LabelFlags.Set(cmd)
	o.TagFlags.Set(cmd)

	return cmd
}

func (po *PushOptions) Run() error {
	reg, err := registry.NewSimpleRegistry(po.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return err
	}

	err = po.validateFlags()
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

	po.ui.BeginLinef("\nPushed: \n%s\n", imageURL)

	return nil
}

func (po *PushOptions) pushBundle(registry registry.Registry) (string, error) {

	imageURL := ""
	imageRefs := []string{}
	uploadRefs := []regname.Tag{}

	baseImageName, err := po.stripTag()
	if err != nil {
		return "", err
	}

	baseRef, err := regname.NewTag(po.BundleFlags.Bundle, regname.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("Parsing '%s': %s", po.BundleFlags.Bundle, err)
	}

	// Append the base image_tag to the list of refs to upload
	uploadRefs = append(uploadRefs, baseRef)

	// TODO(phenixblue): Cleanup when done testing
	fmt.Printf("\nAdding base ref to list of tags: %s\n", baseRef)

	// Loop through all tags specified by the user and push the related image+tag
	for _, tag := range po.TagFlags.Tags {

		uploadRef, err := regname.NewTag(baseImageName+":"+tag, regname.WeakValidation)
		if err != nil {
			return "", fmt.Errorf("Parsing '%s': %s", tag, err)
		}

		uploadRefs = append(uploadRefs, uploadRef)
		// TODO(phenixblue): Cleanup when done testing
		fmt.Printf("\nAdding non-base ref to list of tags: %s\n", uploadRef)

	}

	logger := util.NewUILevelLogger(util.LogWarn, util.NewLogger(po.ui))

	if len(po.TagFlags.Tags) > 1 {
		imageURL, err = bundle.NewContents(po.FileFlags.Files, po.FileFlags.ExcludedFilePaths, po.FileFlags.PreservePermissions).MultiTagPush(uploadRefs, po.LabelFlags.Labels, registry, logger)
		if err != nil {
			return "", err
		}

	} else {
		imageURL, err = bundle.NewContents(po.FileFlags.Files, po.FileFlags.ExcludedFilePaths, po.FileFlags.PreservePermissions).Push(uploadRefs[0], po.LabelFlags.Labels, registry, logger)
		if err != nil {
			return "", err
		}
	}

	if po.LockOutputFlags.LockFilePath != "" {
		bundleLock := lockconfig.BundleLock{
			LockVersion: lockconfig.LockVersion{
				APIVersion: lockconfig.BundleLockAPIVersion,
				Kind:       lockconfig.BundleLockKind,
			},
			Bundle: lockconfig.BundleRef{
				Image:     imageURL,
				Tag:       uploadRefs[0].TagStr(),
				OtherTags: strings.Join(po.TagFlags.Tags, ","),
			},
		}

		err := bundleLock.WriteToPath(po.LockOutputFlags.LockFilePath)
		if err != nil {
			return "", err
		}
	}

	if !strings.Contains(strings.Join(imageRefs, ","), imageURL) {
		imageRefs = append(imageRefs, imageURL)
	}

	po.ui.BeginLinef("\nTags: %s, %s\n", baseRef.TagStr(), strings.Join(po.TagFlags.Tags, ", "))

	return strings.Join(imageRefs, "\n"), nil
}

func (po *PushOptions) pushImage(registry registry.Registry) (string, error) {

	imageURL := ""
	imageRefs := []string{}
	uploadRefs := []regname.Tag{}

	if po.LockOutputFlags.LockFilePath != "" {
		return "", fmt.Errorf("Lock output is not compatible with image, use bundle for lock output")
	}

	isBundle, err := bundle.NewContents(po.FileFlags.Files, po.FileFlags.ExcludedFilePaths, po.FileFlags.PreservePermissions).PresentsAsBundle()
	if err != nil {
		return "", err
	}
	if isBundle {
		return "", fmt.Errorf("Images cannot be pushed with '.imgpkg' directories, consider using --bundle (-b) option")
	}

	baseImageName, err := po.stripTag()
	if err != nil {
		return "", err
	}

	baseRef, err := regname.NewTag(po.ImageFlags.Image, regname.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("Parsing '%s': %s", po.BundleFlags.Bundle, err)
	}

	// Append the base image_tag to the list of refs to upload
	uploadRefs = append(uploadRefs, baseRef)

	// Loop through all tags specified by the user and push the related image+tag
	for _, tag := range po.TagFlags.Tags {

		uploadRef, err := regname.NewTag(baseImageName+":"+tag, regname.WeakValidation)
		if err != nil {
			return "", fmt.Errorf("Parsing '%s': %s", tag, err)
		}

		uploadRefs = append(uploadRefs, uploadRef)

	}

	logger := util.NewUILevelLogger(util.LogWarn, util.NewLogger(po.ui))

	if len(po.TagFlags.Tags) > 1 {
		imageURL, err = plainimage.NewContents(po.FileFlags.Files, po.FileFlags.ExcludedFilePaths, po.FileFlags.PreservePermissions).MultiTagPush(uploadRefs, po.LabelFlags.Labels, registry, logger)
		if err != nil {
			return "", err
		}
	} else {
		imageURL, err = plainimage.NewContents(po.FileFlags.Files, po.FileFlags.ExcludedFilePaths, po.FileFlags.PreservePermissions).Push(uploadRefs[0], po.LabelFlags.Labels, registry, logger)
		if err != nil {
			return "", err
		}
	}

	if !strings.Contains(strings.Join(imageRefs, ","), imageURL) {
		imageRefs = append(imageRefs, imageURL)
	}

	po.ui.BeginLinef("\nTags: %s, %s\n", baseRef.TagStr(), strings.Join(po.TagFlags.Tags, ", "))

	return strings.Join(imageRefs, "\n"), nil
}

// validateFlags checks if the provided flags are valid
func (po *PushOptions) validateFlags() error {

	// Verify the user did NOT specify a reserved OCI label
	_, present := po.LabelFlags.Labels[bundle.BundleConfigLabel]

	if present {
		return fmt.Errorf("label '%s' is reserved and cannot be overriden. Please use a different key", bundle.BundleConfigLabel)
	}

	return nil

}

func (po *PushOptions) stripTag() (string, error) {

	object := ""
	isBundle := po.BundleFlags.Bundle != ""
	isImage := po.ImageFlags.Image != ""

	switch {

	case isBundle:
		object = po.BundleFlags.Bundle

	case isImage:
		object = po.ImageFlags.Image

	default:
		panic("Unreachable code")
	}

	objectRef, err := regname.NewTag(object, regname.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("Parsing '%s': %s", object, err)
	}

	baseObjectName := strings.TrimSuffix(objectRef.Name(), ":"+objectRef.TagStr())

	if baseObjectName == "" {
		return "", fmt.Errorf("'%s' is not a valid image reference", object)
	}

	return baseObjectName, nil
}
