// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type OciFlags struct {
	OciImg string
	OciTar string
}

func (o *OciFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.OciTar, "to-oci-tar", "", "Set OciTarPath to be saved to disk (example: /path/file.tar)")
}
