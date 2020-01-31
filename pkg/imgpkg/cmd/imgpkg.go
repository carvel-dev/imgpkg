package cmd

import (
	"io"

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
	cmd.SetOutput(uiBlockWriter{o.ui}) // setting output for cmd.Help()

	o.UIFlags.Set(cmd)

	cmd.AddCommand(NewPushCmd(NewPushOptions(o.ui)))
	cmd.AddCommand(NewPullCmd(NewPullOptions(o.ui)))
	cmd.AddCommand(NewVersionCmd(NewVersionOptions(o.ui)))

	tagCmd := NewTagCmd()
	tagCmd.AddCommand(NewTagListCmd(NewTagListOptions(o.ui)))
	cmd.AddCommand(tagCmd)

	// Last one runs first
	cobrautil.VisitCommands(cmd, cobrautil.ReconfigureCmdWithSubcmd)
	cobrautil.VisitCommands(cmd, cobrautil.ReconfigureLeafCmd)

	cobrautil.VisitCommands(cmd, cobrautil.WrapRunEForCmd(func(*cobra.Command, []string) error {
		o.UIFlags.ConfigureUI(o.ui)
		return nil
	}))

	cobrautil.VisitCommands(cmd, cobrautil.WrapRunEForCmd(cobrautil.ResolveFlagsForCmd))

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
