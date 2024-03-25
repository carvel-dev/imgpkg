// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type BundleFlags struct {
	Bundle string
}

func (b *BundleFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&b.Bundle, "bundle", "b", "", "Set bundle (example: docker.io/dkalinin/test-content)")
}

func (b *BundleFlags) SetCopy(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&b.Bundle, "bundle", "b", "", "Bundle reference for copying (happens thickly, i.e. bundle image + all referenced images)")
}
