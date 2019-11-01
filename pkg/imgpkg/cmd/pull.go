package cmd

import (
	"fmt"
	"os"

	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/spf13/cobra"
)

type PullOptions struct {
	ui ui.UI

	ImageFlags    ImageFlags
	RegistryFlags RegistryFlags
	OutputPath    string
}

var _ ctlimg.ImagesMetadata = ctlimg.Registry{}

func NewPullOptions(ui ui.UI) *PullOptions {
	return &PullOptions{ui: ui}
}

func NewPullCmd(o *PullOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull files from image",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
		Example: `
  # Pull image dkalinin/app1-config and extract into /tmp/app1-config
  imgpkg pull -i dkalinin/app1-config -o /tmp/app1-config`,
	}
	o.ImageFlags.Set(cmd)
	o.RegistryFlags.Set(cmd)

	cmd.Flags().StringVarP(&o.OutputPath, "output", "o", "", "Output directory path")
	cmd.MarkFlagRequired("output")

	return cmd
}

func (o *PullOptions) Run() error {
	registry := ctlimg.NewRegistry(o.RegistryFlags.AsRegistryOpts())

	ref, err := regname.ParseReference(o.ImageFlags.Image, regname.WeakValidation)
	if err != nil {
		return err
	}

	imgs, err := ctlimg.NewImages(ref, registry).Images()
	if err != nil {
		return fmt.Errorf("Collecting images: %s", err)
	}

	if len(imgs) > 1 {
		o.ui.BeginLinef("Found multiple images, extracting first\n")
	}

	for _, img := range imgs {
		digest, err := img.Digest()
		if err != nil {
			return fmt.Errorf("Getting image digest: %s", err)
		}

		o.ui.BeginLinef("Pulling image '%s@%s'\n", ref.Context(), digest)

		if o.OutputPath == "/" || o.OutputPath == "." || o.OutputPath == ".." {
			return fmt.Errorf("Disallowed output directory (trying to avoid accidental deletion)")
		}

		// TODO protection for destination
		err = os.RemoveAll(o.OutputPath)
		if err != nil {
			return fmt.Errorf("Removing output directory: %s", err)
		}

		err = os.MkdirAll(o.OutputPath, 0700)
		if err != nil {
			return fmt.Errorf("Creating output directory: %s", err)
		}

		err = ctlimg.NewDirImage(o.OutputPath, img, o.ui).AsDirectory()
		if err != nil {
			return fmt.Errorf("Extracting image into directory: %s", err)
		}

		return nil
	}

	return fmt.Errorf("Expected to find at least one image, but found none")
}
