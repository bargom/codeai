package cmd

import (
	"github.com/spf13/cobra"
)

// newCompletionCmd creates the completion command.
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion scripts for CodeAI.

To load completions:

Bash:
  $ source <(codeai completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ codeai completion bash > /etc/bash_completion.d/codeai
  # macOS:
  $ codeai completion bash > /usr/local/etc/bash_completion.d/codeai

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ codeai completion zsh > "${fpath[1]}/_codeai"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ codeai completion fish | source

  # To load completions for each session, execute once:
  $ codeai completion fish > ~/.config/fish/completions/codeai.fish

PowerShell:
  PS> codeai completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> codeai completion powershell > codeai.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(out)
			case "zsh":
				return cmd.Root().GenZshCompletion(out)
			case "fish":
				return cmd.Root().GenFishCompletion(out, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(out)
			}
			return nil
		},
	}

	return cmd
}
