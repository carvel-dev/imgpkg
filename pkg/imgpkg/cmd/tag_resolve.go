// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"carvel.dev/imgpkg/pkg/imgpkg/registry"
	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
)

type TagResolveOptions struct {
	ui ui.UI

	ImageFlags    ImageFlags
	RegistryFlags RegistryFlags
}

func NewTagResolveOptions(ui ui.UI) *TagResolveOptions {
	return &TagResolveOptions{ui: ui}
}

func NewTagResolveCmd(o *TagResolveOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve tag to digest for image",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
	}
	o.ImageFlags.Set(cmd)
	o.RegistryFlags.Set(cmd)
	return cmd
}

func (t *TagResolveOptions) Run() error {
	reg, err := registry.NewSimpleRegistry(t.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return err
	}

	ref, err := regname.ParseReference(t.ImageFlags.Image, regname.WeakValidation)
	if err != nil {
		return err
	}

	digest, err := reg.Digest(ref)
	if err != nil {
		return err
	}

	t.ui.PrintBlock([]byte(fmt.Sprintf("%s@%s", ref.Context(), digest.String())))

	return nil
}
