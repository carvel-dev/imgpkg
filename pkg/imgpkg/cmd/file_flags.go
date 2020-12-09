// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type FileFlags struct {
	Files []string

	ExcludedFileBasenames []string
}

func (s *FileFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringSliceVarP(&s.Files, "file", "f", nil, "Set file (format: /tmp/foo, -) (can be specified multiple times)")

	cmd.Flags().StringSliceVar(&s.ExcludedFileBasenames, "file-exclude-defaults", []string{".git"}, "Excluded file paths by default (can be specified multiple times)")
	cmd.Flags().MarkDeprecated("file-exclude-defaults", "use '--file-base-exclude' instead")

	cmd.Flags().StringSliceVar(&s.ExcludedFileBasenames, "file-base-exclude", []string{".git"}, "Exclude all files whose base name matches (format: bar.yaml) (can be specified multiple times)")
}
