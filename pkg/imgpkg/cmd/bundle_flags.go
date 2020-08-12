// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type BundleFlags struct {
	Bundle string
}

func (s *BundleFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&s.Bundle, "bundle", "b", "", "Set bundle (example: docker.io/dkalinin/test-content)")
}
