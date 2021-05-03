// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:

  $ source <(imgpkg completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ imgpkg completion bash > /etc/bash_completion.d/imgpkg
  # macOS:
  $ imgpkg completion bash > /usr/local/etc/bash_completion.d/imgpkg

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ imgpkg completion zsh > "${fpath[1]}/_imgpkg"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ imgpkg completion fish | source

  # To load completions for each session, execute once:
  $ imgpkg completion fish > ~/.config/fish/completions/imgpkg.fish

PowerShell:

  PS> imgpkg completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> imgpkg completion powershell > imgpkg.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			panic("unreachable")
		},
	}
}
