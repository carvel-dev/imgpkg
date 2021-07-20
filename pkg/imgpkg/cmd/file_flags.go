// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type FileFlags struct {
	Files []string

	ExcludedFilePaths []string
}

func (f *FileFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringSliceVarP(&f.Files, "file", "f", nil, "Set file (format: /tmp/foo) (can be specified multiple times)")

	cmd.Flags().StringSliceVar(&f.ExcludedFilePaths, "file-exclude-defaults", []string{".git"}, "Excluded file paths by default (can be specified multiple times)")
	cmd.Flags().MarkDeprecated("file-exclude-defaults", "use '--file-exclusion' instead")

	cmd.Flags().StringSliceVar(&f.ExcludedFilePaths, "file-exclusion", []string{".git"}, "Exclude file whose path, relative to the bundle root, matches (format: bar.yaml, nested-dir/baz.txt) (can be specified multiple times)")
}
