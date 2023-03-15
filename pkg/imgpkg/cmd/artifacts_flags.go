// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import "github.com/spf13/cobra"

// ArtifactFlags Contains the flags to allow finding and copying cosign artifacts
type ArtifactFlags struct {
	CopyCosignSignatures bool
}

// Set adds the flag to the provided command
func (s *ArtifactFlags) Set(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&s.CopyCosignSignatures, "cosign-artifacts", false, "Find and copy cosign signatures, attestations and sboms for images")
}
