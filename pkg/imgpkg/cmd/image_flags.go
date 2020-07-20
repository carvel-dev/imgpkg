package cmd

import (
	"github.com/spf13/cobra"
)

type ImageFlags struct {
	Image string
}

func (s *ImageFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&s.Image, "image", "i", "", "Set image (example: docker.io/dkalinin/test-content)")
}
