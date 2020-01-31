package cmd

import (
	"fmt"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type PushOptions struct {
	ui ui.UI

	ImageFlags    ImageFlags
	FileFlags     FileFlags
	RegistryFlags RegistryFlags
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
  # Push image dkalinin/app1-config with contents of config/ directory
  imgpkg push -i dkalinin/app1-config -f config/

  # Push image dkalinin/app1-config with contents from multiple locations
  imgpkg push -i dkalinin/app1-config -f config/ -f additional-config.yml`,
	}
	o.ImageFlags.Set(cmd)
	o.FileFlags.Set(cmd)
	o.RegistryFlags.Set(cmd)
	return cmd
}

func (o *PushOptions) Run() error {
	uploadRef, err := regname.NewTag(o.ImageFlags.Image, regname.WeakValidation)
	if err != nil {
		return fmt.Errorf("Parsing image '%s': %s", o.ImageFlags.Image, err)
	}

	registry := ctlimg.NewRegistry(o.RegistryFlags.AsRegistryOpts())

	img, err := ctlimg.NewTarImage(o.FileFlags.Files, InfoLog{o.ui}).AsFileImage()
	if err != nil {
		return err
	}

	defer img.Remove()

	err = registry.WriteImage(uploadRef, img)
	if err != nil {
		return fmt.Errorf("Writing image '%s': %s", uploadRef.Name(), err)
	}

	digest, err := img.Digest()
	if err != nil {
		return err
	}

	imageURL := fmt.Sprintf("%s@%s", uploadRef.Context(), digest)

	o.ui.BeginLinef("Pushed image '%s'\n", imageURL)

	manifest := map[string]interface{}{
		"apiVersion": "imgpkg.k14s.io/v1alpha1",
		"kind":       "PushedImage",
		"image":      imageURL,
	}

	manifestBs, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}

	o.ui.PrintBlock(append([]byte("---\n"), manifestBs...))

	return nil
}
