// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type BundleRecursiveFlags struct {
	Recursive bool
}

func (b *BundleRecursiveFlags) Set(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&b.Recursive, "recursive", "r", false, "Recursively iterate and fetch content of every bundle")
}

func (b *BundleRecursiveFlags) SetCopy(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&b.Recursive, "recursive", "r", false, "Recursively iterate and fetch content of every bundle")
}
