// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type OciFlags struct {
	OcitoReg string
	OciTar   string
}

func (o *OciFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.OciTar, "to-oci-tar", "", "Set OciTarPath to be saved to disk (example: /path/file.tar)")
	cmd.Flags().StringVar(&o.OcitoReg, "oci-tar", "", "Give path to OCI tar file (example: /path/file.tar)")
}

func (o OciFlags) IsOci() bool { return o.OcitoReg != "" }
