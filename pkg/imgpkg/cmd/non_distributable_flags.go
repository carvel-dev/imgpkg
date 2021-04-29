// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import "github.com/spf13/cobra"

type IncludeNonDistributableFlag struct {
	IncludeNonDistributable bool
}

func (i *IncludeNonDistributableFlag) Set(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&i.IncludeNonDistributable, "include-non-distributable-layers", false,
		"Include non-distributable layers when copying an image/bundle")
}
