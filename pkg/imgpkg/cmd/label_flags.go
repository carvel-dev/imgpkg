// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

type LabelFlags struct {
	Labels map[string]string
}

func (l *LabelFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringToStringVarP(&l.Labels, "labels", "l", make(map[string]string), "Set labels on image")
}
