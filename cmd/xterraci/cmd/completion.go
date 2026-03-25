package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd(rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for xterraci.

To load completions:

Bash:
  $ source <(xterraci completion bash)

Zsh:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  $ xterraci completion zsh > "${fpath[1]}/_xterraci"

Fish:
  $ xterraci completion fish | source
  $ xterraci completion fish > ~/.config/fish/completions/xterraci.fish

PowerShell:
  PS> xterraci completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(_ *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
}
