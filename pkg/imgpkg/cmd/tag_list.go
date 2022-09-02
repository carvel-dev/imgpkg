// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/cppforlife/go-cli-ui/ui"
	uitable "github.com/cppforlife/go-cli-ui/ui/table"
	"github.com/spf13/cobra"
	v1 "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/v1"
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
	tagInfo, err := v1.TagList(t.ImageFlags.Image, t.Digests, t.RegistryFlags.AsRegistryOpts())
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

	for _, tag := range tagInfo.Tags {
		table.Rows = append(table.Rows, []uitable.Value{
			uitable.NewValueString(tag.Tag),
			uitable.NewValueString(tag.Digest),
		})
	}

	t.ui.PrintTable(table)

	return nil
}
