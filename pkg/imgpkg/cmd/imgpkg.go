// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"io"

	"github.com/cppforlife/cobrautil"
	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/spf13/cobra"
)

type ImgpkgOptions struct {
	ui *ui.ConfUI

	UIFlags    UIFlags
	DebugFlags DebugFlags
}

func NewImgpkgOptions(ui *ui.ConfUI) *ImgpkgOptions {
	return &ImgpkgOptions{ui: ui}
}

func NewDefaultImgpkgCmd(ui *ui.ConfUI) *cobra.Command {
	return NewImgpkgCmd(NewImgpkgOptions(ui))
}

func NewImgpkgCmd(o *ImgpkgOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "imgpkg",
		Short:             "imgpkg allows to store configuration and image references as OCI artifacts",
		SilenceErrors:     true,
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		Version:           Version,
	}

	// TODO bash completion
	// setting output for cmd.Help()
	blockWriter := uiBlockWriter{o.ui}
	cmd.SetOut(blockWriter)
	cmd.SetErr(blockWriter)

	o.UIFlags.Set(cmd)
	o.DebugFlags.Set(cmd)

	cmd.AddCommand(NewPushCmd(NewPushOptions(o.ui)))
	cmd.AddCommand(NewPullCmd(NewPullOptions(o.ui)))
	cmd.AddCommand(NewVersionCmd(NewVersionOptions(o.ui)))
	cmd.AddCommand(NewCopyCmd(NewCopyOptions(o.ui)))

	tagCmd := NewTagCmd()
	tagCmd.AddCommand(NewTagListCmd(NewTagListOptions(o.ui)))
	tagCmd.AddCommand(NewTagResolveCmd(NewTagResolveOptions(o.ui)))
	cmd.AddCommand(tagCmd)

	// Last one runs first
	cobrautil.VisitCommands(cmd, cobrautil.ReconfigureCmdWithSubcmd)
	cobrautil.VisitCommands(cmd, cobrautil.DisallowExtraArgs)

	cobrautil.VisitCommands(cmd, cobrautil.WrapRunEForCmd(func(*cobra.Command, []string) error {
		o.UIFlags.ConfigureUI(o.ui)
		o.DebugFlags.ConfigureDebug()
		return nil
	}))

	cobrautil.VisitCommands(cmd, cobrautil.WrapRunEForCmd(cobrautil.ResolveFlagsForCmd))

	// Completion command have to be added after the VisitCommands
	// This due to the ReconfigureLeafCmds that we do not want to have enforced for the completion
	// This configurations forces all nodes to do not accept extra args, but the completion requires 1 extra arg
	cmd.AddCommand(NewCompletionCmd())

	return cmd
}

type uiBlockWriter struct {
	ui ui.UI
}

var _ io.Writer = uiBlockWriter{}

func (w uiBlockWriter) Write(p []byte) (n int, err error) {
	w.ui.PrintBlock(p)
	return len(p), nil
}
