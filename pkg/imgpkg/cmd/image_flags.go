// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type ImageFlags struct {
	Image  string
	Labels map[string]string
}

func (i *ImageFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&i.Image, "image", "i", "", "Set image (example: docker.io/dkalinin/test-content)")
	cmd.Flags().StringToStringVarP(&i.Labels, "labels", "l", map[string]string{}, "Set labels on image")
}

func (i *ImageFlags) SetCopy(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&i.Image, "image", "i", "", "Image reference for copying a generic image (example: docker.io/dkalinin/test-content)")
}
