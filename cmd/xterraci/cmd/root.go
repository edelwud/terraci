// Package cmd provides the xterraci CLI commands.
package cmd

import "github.com/spf13/cobra"

// App holds shared state for all xterraci commands.
type App struct {
	Version string
	Commit  string
	Date    string
}

// NewRootCmd creates the xterraci root command with all subcommands.
func NewRootCmd(version, commit, date string) *cobra.Command {
	app := &App{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	rootCmd := &cobra.Command{
		Use:   "xterraci",
		Short: "Build custom TerraCi binaries with plugins",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.AddCommand(newBuildCmd(app))
	rootCmd.AddCommand(newListPluginsCmd())
	rootCmd.AddCommand(newVersionCmd(app))
	rootCmd.AddCommand(newCompletionCmd(rootCmd))
	rootCmd.AddCommand(newManCmd(rootCmd))

	return rootCmd
}
