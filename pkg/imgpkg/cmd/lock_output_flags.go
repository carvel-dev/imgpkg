// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type LockOutputFlags struct {
	LockFilePath string
}

func (s *LockOutputFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVar(&s.LockFilePath, "lock-output", "",
		"Location to output lockfile. lockfile type (BundleLock or ImagesLock) is determined from contents moved.")
}
