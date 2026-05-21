package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
)

// NewRootCmd creates and returns the root cobra command with all subcommands.
func NewRootCmd(version, commit, date string) *cobra.Command {
	app := newApp(version, commit, date)

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	rootCmd := &cobra.Command{
		Use:           "terraci",
		Short:         "Generate CI pipelines for Terraform projects",
		SilenceUsage:  true,
		SilenceErrors: true,
		Long: `TerraCi is a CLI tool that analyzes Terraform project structure,
builds a dependency graph based on terraform_remote_state references,
and generates CI pipelines (GitLab CI or GitHub Actions) that respect those dependencies.

Features:
  - Automatic discovery of Terraform modules
  - Dependency graph from terraform_remote_state
  - Support for for_each in remote state
  - Glob pattern filtering for modules
  - Git integration for changed-only pipelines
  - Parallel execution where possible`,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			verbose, verboseErr := cmd.Flags().GetBool("verbose")
			if verboseErr != nil {
				verbose = false
			}
			result, err := app.newRunFlow().Prepare(cmd.Context(), runflow.Request{
				CommandName: cmd.Name(),
				ConfigPath:  app.cfgFile,
				WorkDir:     app.WorkDir,
				LogLevel:    app.logLevel,
				Verbose:     verbose,
				Policy:      runflow.PolicyFromCommand(cmd),
			})
			if err != nil {
				return err
			}
			app.reports = result.Reports()
			cmd.SetContext(result.Context())
			return nil
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&app.cfgFile, "config", "c", "", "config file (default: .terraci.yaml)")
	rootCmd.PersistentFlags().StringVarP(&app.WorkDir, "dir", "d", cwd, "working directory")
	rootCmd.PersistentFlags().StringVarP(&app.logLevel, "log-level", "l", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output (shorthand for --log-level=debug)")

	// Register core subcommands
	rootCmd.AddCommand(newGenerateCmd())
	rootCmd.AddCommand(newGraphCmd())
	rootCmd.AddCommand(newValidateCmd())
	// Note: summary and policy commands are now provided by plugins
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newVersionCmd(app))
	rootCmd.AddCommand(newSchemaCmd())
	rootCmd.AddCommand(newCompletionCmd(rootCmd))
	rootCmd.AddCommand(newManCmd(rootCmd))

	// Register plugin-provided commands. Commands() runs at registration
	// time and must not capture state — plugins retrieve the per-run
	// AppContext and command-scoped plugin inside RunE via plugin.CommandPlugin.
	for _, cmd := range runflow.PluginCommands(nil) {
		rootCmd.AddCommand(cmd)
	}

	return rootCmd
}
