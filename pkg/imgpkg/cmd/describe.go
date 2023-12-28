// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"sort"

	"carvel.dev/imgpkg/pkg/imgpkg/bundle"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	v1 "carvel.dev/imgpkg/pkg/imgpkg/v1"
	goui "github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
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

	Concurrency            int
	OutputType             string
	IncludeCosignArtifacts bool
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
	cmd.Flags().BoolVar(&o.IncludeCosignArtifacts, "cosign-artifacts", true, "Retrieve cosign artifact information (Default: true)")
	return cmd
}

// Run functions called when the describe command is provided in the command line
func (d *DescribeOptions) Run() error {
	err := d.validateFlags()
	if err != nil {
		return err
	}
	logLevel := util.LogWarn

	levelLogger := util.NewUILevelLogger(logLevel, util.NewLogger(d.ui))
	description, err := v1.Describe(
		d.BundleFlags.Bundle,
		v1.DescribeOpts{
			Logger:                 levelLogger,
			Concurrency:            d.Concurrency,
			IncludeCosignArtifacts: d.IncludeCosignArtifacts,
		},
		d.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return err
	}

	ttyEnabledLogger := util.NewUILevelLogger(logLevel, util.NewLoggerNoTTY(d.ui))
	if d.OutputType == "text" {
		p := bundleTextPrinter{logger: ttyEnabledLogger}
		p.Print(description)
	} else if d.OutputType == "yaml" {
		p := bundleYAMLPrinter{logger: ttyEnabledLogger}
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
	logger Logger
}

func (p bundleTextPrinter) Print(description v1.Description) {
	bundleRef, err := regname.ParseReference(description.Image)
	if err != nil {
		panic(fmt.Sprintf("Internal consistency: expected %s to be a digest reference", description.Image))
	}
	p.logger.Logf("Bundle SHA: %s\n", bundleRef.Identifier())

	p.logger.Logf("\n")
	p.printerRec(description, p.logger, p.logger)
}

func (p bundleTextPrinter) printerRec(description v1.Description, originalLogger Logger, logger Logger) {
	indentLogger := util.NewIndentedLogger(logger)
	if len(description.Content.Bundles) == 0 && len(description.Content.Images) == 0 {
		return
	}
	if originalLogger == logger {
		originalLogger.Logf("Images:\n")
	} else {
		indentLogger.Logf("Images:\n")
	}
	firstBundle := true
	for _, b := range description.Content.Bundles {
		if !firstBundle {
			originalLogger.Logf("\n")
		} else {
			firstBundle = false
		}
		indentLogger.Logf("- Image: %s\n", b.Image)
		indentLogger.Logf("  Type: Bundle\n")
		indentLogger.Logf("  Origin: %s\n", b.Origin)
		indentLogger.Logf("  Layers:\n")
		for _, d := range b.Layers {
			indentLogger.Logf("    - Digest: %s\n", d)
		}
		annotations := b.Annotations

		p.printAnnotations(annotations, util.NewIndentedLogger(indentLogger))
		p.printerRec(b, originalLogger, indentLogger)
	}

	if len(description.Content.Bundles) > 0 {
		originalLogger.Logf("")
	}

	firstImage := true
	for _, image := range description.Content.Images {
		if !firstImage {
			originalLogger.Logf("")
		} else {
			firstImage = false
		}
		indentLogger.Logf("- Image: %s\n", image.Image)
		indentLogger.Logf("  Type: %s\n", image.ImageType)
		if image.ImageType == bundle.ContentImage {
			indentLogger.Logf("  Origin: %s\n", image.Origin)
		}
		indentLogger.Logf("  Layers:\n")
		for _, d := range image.Layers {
			indentLogger.Logf("    - Digest: %s\n", d)
		}
		annotations := image.Annotations
		p.printAnnotations(annotations, util.NewIndentedLogger(indentLogger))
	}
}

func (p bundleTextPrinter) printAnnotations(annotations map[string]string, indentLogger Logger) {
	if len(annotations) > 0 {
		indentLogger.Logf("Annotations:\n")
		annIndentLogger := util.NewIndentedLogger(indentLogger)

		var annotationKeys []string
		for key := range annotations {
			annotationKeys = append(annotationKeys, key)
		}
		sort.Strings(annotationKeys)
		for _, key := range annotationKeys {
			annIndentLogger.Logf("%s: %s\n", key, annotations[key])
		}
	}
}

type bundleYAMLPrinter struct {
	logger Logger
}

func (p bundleYAMLPrinter) Print(description v1.Description) error {
	bundleRef, err := regname.ParseReference(description.Image)
	if err != nil {
		panic(fmt.Sprintf("Internal consistency: expected %s to be a digest reference", description.Image))
	}

	yamlDesc, err := yaml.Marshal(description)
	if err != nil {
		return err
	}

	p.logger.Logf("sha: %s\n", bundleRef.Identifier())
	p.logger.Logf(string(yamlDesc))

	return nil
}
