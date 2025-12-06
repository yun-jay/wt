package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var completionsCmd = &cobra.Command{
	Use:   "completions <shell>",
	Short: "Generate shell completions",
	Long: `Generate shell completions for wt.

Supported shells: bash, zsh, fish, powershell

Add to your shell config:

Bash (~/.bashrc):
  eval "$(wt completions bash)"

Zsh (~/.zshrc):
  eval "$(wt completions zsh)"

Fish (~/.config/fish/config.fish):
  wt completions fish | source

PowerShell:
  wt completions powershell | Out-String | Invoke-Expression`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE:      runCompletions,
}

func runCompletions(cmd *cobra.Command, args []string) error {
	shell := args[0]

	switch shell {
	case "bash":
		return rootCmd.GenBashCompletion(os.Stdout)
	case "zsh":
		return rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		return rootCmd.GenFishCompletion(os.Stdout, true)
	case "powershell":
		return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return cmd.Help()
	}
}
