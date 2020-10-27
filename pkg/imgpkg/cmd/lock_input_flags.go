// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type LockInputFlags struct {
	LockFilePath string
}

func (s *LockInputFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVar(&s.LockFilePath, "lock", "",
		"Lock file with asset references to copy to destination")
}
