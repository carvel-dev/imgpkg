// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type LockInputFlags struct {
	LockFilePath string
}

func (l *LockInputFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVar(&l.LockFilePath, "lock", "",
		"Lock file with asset references to copy to destination")
}
