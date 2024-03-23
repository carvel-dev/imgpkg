// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"

	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/spf13/cobra"
)

// DebugFlags indicates debugging
type DebugFlags struct {
	Debug bool
}

// Set adds the debug flag to the command
func (f *DebugFlags) Set(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&f.Debug, "debug", false, "Enables debugging")
}

// ConfigureDebug set debug output to os.Stdout
func (f *DebugFlags) ConfigureDebug() {
	if f.Debug {
		logs.Debug.SetOutput(os.Stderr)
	}
}
