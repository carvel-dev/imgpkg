package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/google/go-containerregistry/pkg/name"
	regname "github.com/google/go-containerregistry/pkg/name"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type PushOptions struct {
	ui ui.UI

	ImageFlags    ImageFlags
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

  # Push image dkalinin/app1-config with contents from multiple locations
  imgpkg push -i dkalinin/app1-config -f config/ -f additional-config.yml`,
	}
	o.ImageFlags.Set(cmd)
	o.BundleFlags.Set(cmd)
	o.OutputFlags.Set(cmd)
	o.FileFlags.Set(cmd)
	o.RegistryFlags.Set(cmd)
	return cmd
}

func (o *PushOptions) Run() error {
	err := o.validateFlags()
	if err != nil {
		return err
	}

	bundleDirPath, err := o.extractBundleDirPath()
	if err != nil {
		return err
	}

	if o.isBundle() {
		o.validateImages(bundleDirPath)
	}

	err = o.checkRepeatedPaths()
	if err != nil {
		return err
	}

	var inputRef string
	if o.isBundle() {
		inputRef = o.BundleFlags.Bundle
	} else {
		inputRef = o.ImageFlags.Image
	}

	uploadRef, err := regname.NewTag(inputRef, regname.WeakValidation)
	if err != nil {
		return fmt.Errorf("Parsing '%s': %s", inputRef, err)
	}

	registry := ctlimg.NewRegistry(o.RegistryFlags.AsRegistryOpts())

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

	if o.OutputFlags.LockFilePath != "" {
		bundleLock := BundleLock{
			ApiVersion: "imgpkg.k14s.io/v1alpha1",
			Kind:       "BundleLock",
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

		err = ioutil.WriteFile(o.OutputFlags.LockFilePath, append([]byte("---\n"), manifestBs...), 0700)
		if err != nil {
			return fmt.Errorf("Writing lock file: %s", err)
		}
	}

	return nil
}

// TODO rename when we have a name
const BundleDir = ".imgpkg"

func (o *PushOptions) validateFlags() error {
	if o.isImage() {
		if o.isBundle() {
			return fmt.Errorf("Expected only one of image or bundle")
		}

		if o.OutputFlags.LockFilePath != "" {
			return fmt.Errorf("Lock output is not compatible with image, use bundle for lock output")
		}
	}

	if !o.isImage() && !o.isBundle() {
		return fmt.Errorf("Expected either image or bundle")
	}

	return nil
}

func (o *PushOptions) validateImages(bundleDirPath string) error {
	imagesBytes, err := ioutil.ReadFile(filepath.Join(bundleDirPath, "images.yml"))
	if err != nil {
		return err
	}

	var imagesLock ImageLock
	err = yaml.Unmarshal(imagesBytes, &imagesLock)
	if err != nil {
		return err
	}

	for _, image := range imagesLock.Spec.Images {
		if _, err := name.NewDigest(image.DigestRef); err != nil {
			return errors.Errorf("Expected ref to be in digest form, got %s", image.DigestRef)
		}
	}
	return nil
}

func (o *PushOptions) extractBundleDirPath() (string, error) {
	var bundlePaths []string
	for _, flagPath := range o.FileFlags.Files {
		err := filepath.Walk(flagPath, func(currPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if filepath.Base(currPath) != BundleDir {
				return nil
			}

			if o.isImage() {
				return fmt.Errorf("Images cannot be pushed with a '%s' bundle directory (found at '%s'), consider using a bundle", BundleDir, currPath)
			}

			if filepath.Dir(currPath) != flagPath {
				return fmt.Errorf("Expected '%s' dir to be a direct child of '%s', but was: '%s'", BundleDir, flagPath, currPath)
			}

			currPath, err = filepath.Abs(currPath)
			if err != nil {
				return err
			}

			bundlePaths = append(bundlePaths, currPath)

			return nil
		})

		if err != nil {
			return "", err
		}
	}

	switch {
	case len(bundlePaths) == 0 && o.isImage():
		return "", nil
	case len(bundlePaths) == 1:
		return bundlePaths[0], nil
	default:
		return "", fmt.Errorf("Expected one '%s' dir, got %d: %s", BundleDir, len(bundlePaths), strings.Join(bundlePaths, ", "))
	}
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
