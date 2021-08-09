// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type TarFlags struct {
	TarSrc string
	TarDst string
}

func (t *TarFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVar(&t.TarDst, "to-tar", "", "Location to write a tar file containing assets")
	cmd.Flags().StringVar(&t.TarSrc, "tar", "", "Path to tar file which contains assets to be copied to a registry")
}

func (t TarFlags) IsSrc() bool { return t.TarSrc != "" }
func (t TarFlags) IsDst() bool { return t.TarDst != "" }
