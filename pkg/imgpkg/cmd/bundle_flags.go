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
