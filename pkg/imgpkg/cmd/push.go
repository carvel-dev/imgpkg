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

type PushOptions struct {
	ui ui.UI

	BundleFlags   BundleFlags
	OutputFlags   OutputFlags
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
  # Push bundle dkalinin/app1-config with contents of config/ directory
  imgpkg push -b dkalinin/app1-config -f config/

  # Push bundle dkalinin/app1-config with contents from multiple locations
  imgpkg push -b dkalinin/app1-config -f config/ -f additional-config.yml`,
	}
	o.OutputFlags.Set(cmd)
	o.BundleFlags.Set(cmd)
	o.FileFlags.Set(cmd)
	o.RegistryFlags.Set(cmd)
	return cmd
}

func (o *PushOptions) Run() error {
	err := o.validateFiles()
	if err != nil {
		return err
	}

	uploadRef, err := regname.NewTag(o.BundleFlags.Bundle, regname.WeakValidation)
	if err != nil {
		return fmt.Errorf("Parsing bundle '%s': %s", o.BundleFlags.Bundle, err)
	}

	registry := ctlimg.NewRegistry(o.RegistryFlags.AsRegistryOpts())

	img, err := ctlimg.NewTarImage(o.FileFlags.Files, o.FileFlags.FileExcludeDefaults, InfoLog{o.ui}).AsFileImage()
	if err != nil {
		return err
	}

	defer img.Remove()

	err = registry.WriteImage(uploadRef, img)
	if err != nil {
		return fmt.Errorf("Writing bundle '%s': %s", uploadRef.Name(), err)
	}

	digest, err := img.Digest()
	if err != nil {
		return err
	}

	imageURL := fmt.Sprintf("%s@%s", uploadRef.Context(), digest)

	o.ui.BeginLinef("Pushed bundle '%s'", imageURL)

	if o.OutputFlags.LockFilePath != "" {
		bundleLock := BundleLock{
			ApiVersion: "imgpkg.k14s.io/v1alpha1",
			Kind:       "BundleLock",
			Spec: BundleSpec{
				Image: BundleImage{
					Url: imageURL,
					Tag: uploadRef.TagStr(),
				},
			},
		}

		manifestBs, err := yaml.Marshal(bundleLock)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(o.OutputFlags.LockFilePath, append([]byte("---\n"), manifestBs...), 0700)
		if err != nil {
			return fmt.Errorf("Writing lock file: %s", err)
		}
	}

	return nil
}

// TODO rename when we have a name
const BundleDir = ".imgpkg"

func (o *PushOptions) validateFiles() error {
	var bundlePaths []string
	for _, inputPath := range o.FileFlags.Files {
		fi, err := os.Stat(inputPath)
		if err != nil {
			return err
		}

		if !fi.IsDir() {
			continue
		}

		err = filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if filepath.Base(path) != BundleDir {
				return nil
			}

			if filepath.Dir(path) != inputPath {
				return fmt.Errorf("Expected '%s' dir to be a direct child of '%s', but was: '%s'", BundleDir, inputPath, path)
			}

			path, err = filepath.Abs(path)
			if err != nil {
				return err
			}

			bundlePaths = append(bundlePaths, path)

			return nil
		})

		if err != nil {
			return err
		}
	}

	if len(bundlePaths) > 1 {
		return fmt.Errorf("Expected one '%s' dir, got %d: %s", BundleDir, len(bundlePaths), strings.Join(bundlePaths, ", "))
	}

	return nil
}
