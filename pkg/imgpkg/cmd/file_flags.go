// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type FileFlags struct {
	Files []string

	FileExcludeDefaults []string
}

func (s *FileFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringSliceVarP(&s.Files, "file", "f", nil, "Set file (format: /tmp/foo, -) (can be specified multiple times)")

	cmd.Flags().StringSliceVar(&s.FileExcludeDefaults, "file-exclude-defaults", []string{".git"}, "Excluded file paths by default (can be specified multiple times)")
}
