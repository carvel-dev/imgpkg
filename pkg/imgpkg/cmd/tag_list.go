// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/cppforlife/go-cli-ui/ui"
	uitable "github.com/cppforlife/go-cli-ui/ui/table"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
)

type TagListOptions struct {
	ui ui.UI

	ImageFlags    ImageFlags
	RegistryFlags RegistryFlags
	Digests       bool
}

func NewTagListOptions(ui ui.UI) *TagListOptions {
	return &TagListOptions{ui: ui}
}

func NewTagListCmd(o *TagListOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List tags for image",
		RunE:    func(_ *cobra.Command, _ []string) error { return o.Run() },
	}
	o.ImageFlags.Set(cmd)
	o.RegistryFlags.Set(cmd)
	// Too slow to resolve each tag to digest individually (no bulk API).
	cmd.Flags().BoolVar(&o.Digests, "digests", false, "Include digests")
	return cmd
}

func (t *TagListOptions) Run() error {
	reg, err := registry.NewSimpleRegistry(t.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return err
	}

	ref, err := regname.ParseReference(t.ImageFlags.Image, regname.WeakValidation)
	if err != nil {
		return err
	}

	tags, err := reg.ListTags(ref.Context())
	if err != nil {
		return err
	}

	digestHeader := uitable.NewHeader("Digest")
	digestHeader.Hidden = !t.Digests

	table := uitable.Table{
		Title:   "Tags",
		Content: "tags",

		Header: []uitable.Header{
			uitable.NewHeader("Name"),
			digestHeader,
		},

		SortBy: []uitable.ColumnSort{
			{Column: 0, Asc: true},
		},
	}

	for _, tag := range tags {
		var digest string

		if t.Digests {
			tagRef, err := regname.NewTag(ref.Context().String()+":"+tag, regname.WeakValidation)
			if err != nil {
				return err
			}

			hash, err := reg.Digest(tagRef)
			if err != nil {
				return err
			}

			digest = hash.String()
		}

		table.Rows = append(table.Rows, []uitable.Value{
			uitable.NewValueString(tag),
			uitable.NewValueString(digest),
		})
	}

	t.ui.PrintTable(table)

	return nil
}
