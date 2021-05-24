// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import "github.com/spf13/cobra"

type SignatureFlags struct {
	CopyCosignSignatures bool
}

func (s *SignatureFlags) Set(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&s.CopyCosignSignatures, "cosign-signatures", false, "Find and copy cosign signatures for images")
}
