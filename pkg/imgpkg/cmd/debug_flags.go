// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/spf13/cobra"
	"os"
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
