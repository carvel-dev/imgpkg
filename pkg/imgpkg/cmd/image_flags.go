// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

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

func (s *ImageFlags) SetCopy(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&s.Image, "image", "i", "", "Image reference for copying a generic image (example: docker.io/dkalinin/test-content)")
}
