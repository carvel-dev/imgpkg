// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type BundleFlags struct {
	Bundle  string
	Recurse bool
}

func (s *BundleFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&s.Bundle, "bundle", "b", "", "Set bundle (example: docker.io/dkalinin/test-content)")
	cmd.Flags().BoolVarP(&s.Recurse, "recursive", "r", false, "Recursively iterate and fetch content of every bundle")
}

func (s *BundleFlags) SetCopy(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&s.Bundle, "bundle", "b", "", "Bundle reference for copying (happens thickly, i.e. bundle image + all referenced images)")
	cmd.Flags().BoolVarP(&s.Recurse, "recursive", "r", false, "Recursively iterate and fetch content of every bundle")
}
