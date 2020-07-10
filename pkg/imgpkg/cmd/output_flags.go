package cmd

import (
	"github.com/spf13/cobra"
)

type OutputFlags struct {
	LockFilePath string
}

func (s *OutputFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVar(&s.LockFilePath, "lock-output", "",
		"Path to output a lock file to (either BundleLock or ImageLock)")
}
