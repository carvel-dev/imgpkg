// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

// TagFlags is a struct that holds the additional tags for an OCI artifact
type TagFlags struct {
	Tags []string
}

// Set sets additional tags for an OCI artifact
func (t *TagFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&t.Tags, "additional-tags", []string{}, "Set additional tags on image")
}
