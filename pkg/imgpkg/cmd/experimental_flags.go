// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import "github.com/spf13/cobra"

type ExperimentalFlags struct {
	RecursiveBundles bool
}

func (s *ExperimentalFlags) Set(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&s.RecursiveBundles, "experimental-recursive-bundle", false, "Enable the experimental functionality of Bundles of Bundles")
}
