// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"regexp"

	"github.com/cppforlife/cobrautil"
	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/spf13/cobra"
)

type ImgpkgOptions struct {
	ui *ui.ConfUI

	UIFlags UIFlags
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
		Short:             "imgpkg stores files as Docker images",
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

	cmd.AddCommand(NewPushCmd(NewPushOptions(o.ui)))
	cmd.AddCommand(NewPullCmd(NewPullOptions(o.ui)))
	cmd.AddCommand(NewVersionCmd(NewVersionOptions(o.ui)))
	cmd.AddCommand(NewCopyCmd(NewCopyOptions()))

	tagCmd := NewTagCmd()
	tagCmd.AddCommand(NewTagListCmd(NewTagListOptions(o.ui)))
	tagCmd.AddCommand(NewTagResolveCmd(NewTagResolveOptions(o.ui)))
	cmd.AddCommand(tagCmd)

	// Last one runs first
	cobrautil.VisitCommands(cmd, cobrautil.ReconfigureCmdWithSubcmd)
	cobrautil.VisitCommands(cmd, cobrautil.DisallowExtraArgs)

	cobrautil.VisitCommands(cmd, cobrautil.WrapRunEForCmd(func(*cobra.Command, []string) error {
		o.UIFlags.ConfigureUI(o.ui)
		return nil
	}))

	cobrautil.VisitCommands(cmd, cobrautil.WrapRunEForCmd(cobrautil.ResolveFlagsForCmd))

	cobrautil.VisitCommands(cmd, cobrautil.WrapRunEForCmd(func(cmd *cobra.Command, args []string) error {
		protocolMatcher := regexp.MustCompile(`^https*://`)

		bundleFlag := cmd.Flag("bundle")
		if bundleFlag != nil && protocolMatcher.MatchString(bundleFlag.Value.String()) {
			return fmt.Errorf("bundle flag %v starts with protocol: remove http(s):// from bundle", bundleFlag.Value)
		}

		imageFlag := cmd.Flag("image")
		if imageFlag != nil && protocolMatcher.MatchString(imageFlag.Value.String()) {
			return fmt.Errorf("image flag %v starts with protocol: remove http(s):// from image", imageFlag.Value)
		}
		return nil
	}))

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
