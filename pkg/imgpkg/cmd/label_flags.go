// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

// LabelFlags is a struct that holds the labels for an OCI artifact
type LabelFlags struct {
	Labels map[string]string
}

// Set sets the labels for an OCI artifact
func (l *LabelFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringToStringVarP(&l.Labels, "labels", "l", map[string]string{}, "Set labels on image")
}
