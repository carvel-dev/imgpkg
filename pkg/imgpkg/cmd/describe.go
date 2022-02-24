// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sort"

	goui "github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
	"sigs.k8s.io/yaml"
)

var (
	// DescribeOutputType Possible output options
	DescribeOutputType = []string{"text", "yaml"}
)

// DescribeOptions Command Line options that can be provided to the describe command
type DescribeOptions struct {
	ui goui.UI

	BundleFlags   BundleFlags
	RegistryFlags RegistryFlags

	Concurrency int
	OutputType  string
}

// NewDescribeOptions constructor for building a DescribeOptions, holding values derived via flags
func NewDescribeOptions(ui *goui.ConfUI) *DescribeOptions {
	return &DescribeOptions{ui: ui}
}

// NewDescribeCmd constructor for the describe command
func NewDescribeCmd(o *DescribeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe the images and bundles associated with a give bundle",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
		Example: `
    # Describe a bundle
    imgpkg describe -b carvel.dev/app1-bundle`,
	}

	o.BundleFlags.SetCopy(cmd)
	o.RegistryFlags.Set(cmd)
	cmd.Flags().IntVar(&o.Concurrency, "concurrency", 5, "Concurrency")
	cmd.Flags().StringVarP(&o.OutputType, "output-type", "o", "text", "Type of output possible values: [text, yaml]")
	return cmd
}

// Run functions called when the describe command is provided in the command line
func (d *DescribeOptions) Run() error {
	err := d.validateFlags()
	if err != nil {
		return err
	}

	levelLogger := util.NewUILevelLogger(util.LogWarn, d.ui)
	description, err := bundle.Describe(
		d.BundleFlags.Bundle,
		bundle.DescribeOpts{
			Logger:      levelLogger,
			Concurrency: d.Concurrency,
		},
		d.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return err
	}

	if d.OutputType == "text" {
		p := bundleTextPrinter{ui: d.ui}
		p.Print(description)
	} else if d.OutputType == "yaml" {
		p := bundleYAMLPrinter{ui: d.ui}
		return p.Print(description)
	}
	return nil
}

func (d *DescribeOptions) validateFlags() error {
	outputType := ""
	for _, s := range DescribeOutputType {
		if s == d.OutputType {
			outputType = s
			break
		}
	}
	if outputType == "" {
		return fmt.Errorf("--output-type can only have the following values [text, yaml]")
	}
	return nil
}

type bundleTextPrinter struct {
	ui goui.UI
}

func (p bundleTextPrinter) Print(description bundle.Description) {
	logger := util.NewUIPrefixedWriter("", p.ui)
	bundleRef, err := regname.ParseReference(description.Image)
	if err != nil {
		panic(fmt.Sprintf("Internal consistency: expected %s to be a digest reference", description.Image))
	}
	logger.BeginLinef("Bundle SHA: %s\n", bundleRef.Identifier())

	logger.BeginLinef("\n")
	p.printerRec(description, logger, logger)
}

func (p bundleTextPrinter) printerRec(description bundle.Description, originalLogger goui.UI, logger goui.UI) {
	indentLogger := goui.NewIndentingUI(logger)
	if len(description.Content.Bundles) == 0 && len(description.Content.Images) == 0 {
		return
	}
	if originalLogger == logger {
		originalLogger.BeginLinef("Images:\n")
	} else {
		indentLogger.BeginLinef("Images:\n")
	}
	for i, b := range description.Content.Bundles {
		if i != 0 {
			originalLogger.BeginLinef("\n")
		}
		indentLogger.BeginLinef("- Image: %s\n", b.Image)
		indentLogger.BeginLinef("  Type: Bundle\n")
		indentLogger.BeginLinef("  Origin: %s\n", b.Origin)
		annotations := b.Annotations

		p.printAnnotations(annotations, goui.NewIndentingUI(indentLogger))
		p.printerRec(b, originalLogger, indentLogger)
	}

	if len(description.Content.Bundles) > 0 {
		originalLogger.BeginLinef("")
	}

	for i, image := range description.Content.Images {
		if i != 0 {
			originalLogger.BeginLinef("")
		}
		indentLogger.BeginLinef("- Image: %s\n", image.Image)
		indentLogger.BeginLinef("  Type: %s\n", image.ImageType)
		if image.ImageType == bundle.ContentImage {
			indentLogger.BeginLinef("  Origin: %s\n", image.Origin)
		}
		annotations := image.Annotations
		p.printAnnotations(annotations, goui.NewIndentingUI(indentLogger))
	}
}

func (p bundleTextPrinter) printAnnotations(annotations map[string]string, indentLogger *goui.IndentingUI) {
	if len(annotations) > 0 {
		indentLogger.BeginLinef("Annotations:\n")
		annIndentLogger := goui.NewIndentingUI(indentLogger)

		var annotationKeys []string
		for key := range annotations {
			annotationKeys = append(annotationKeys, key)
		}
		sort.Strings(annotationKeys)
		for _, key := range annotationKeys {
			annIndentLogger.BeginLinef("%s: %s\n", key, annotations[key])
		}
	}
}

type bundleYAMLPrinter struct {
	ui goui.UI
}

func (p bundleYAMLPrinter) Print(description bundle.Description) error {
	logger := util.NewUIPrefixedWriter("", p.ui)
	bundleRef, err := regname.ParseReference(description.Image)
	if err != nil {
		panic(fmt.Sprintf("Internal consistency: expected %s to be a digest reference", description.Image))
	}

	yamlDesc, err := yaml.Marshal(description)
	if err != nil {
		return err
	}

	logger.BeginLinef("sha: %s\n", bundleRef.Identifier())
	logger.BeginLinef(string(yamlDesc))

	return nil
}
