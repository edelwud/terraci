package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

// NewRootCmd creates and returns the root cobra command with all subcommands.
func NewRootCmd(version, commit, date string) *cobra.Command {
	app := &App{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

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
			log.Init()

			verbose, verboseErr := cmd.Flags().GetBool("verbose")
			if verboseErr == nil && verbose {
				app.logLevel = "debug"
			}

			if app.logLevel != "" {
				if levelErr := log.SetLevelFromString(app.logLevel); levelErr != nil {
					return fmt.Errorf("invalid log level %q: %w", app.logLevel, levelErr)
				}
			}

			if cmd.Name() != "version" && app.Version != "" {
				log.WithField("version", app.Version).Debug("terraci")
			}

			// Skip config loading for commands that don't need it (marked with annotation)
			if cmd.Annotations["skipConfig"] == "true" {
				return nil
			}

			log.Debug("loading configuration")
			var loadErr error
			if app.cfgFile != "" {
				log.WithField("file", app.cfgFile).Debug("loading config from file")
				app.Config, loadErr = config.Load(app.cfgFile)
			} else {
				log.WithField("dir", app.WorkDir).Debug("loading config from directory")
				app.Config, loadErr = config.LoadOrDefault(app.WorkDir)
			}

			if loadErr != nil {
				return loadErr
			}

			log.Debug("validating configuration")
			if err := app.Config.Validate(); err != nil {
				return err
			}

			// Initialize plugin configs
			log.Debug("initializing plugin configurations")
			if err := app.InitPluginConfigs(); err != nil {
				return err
			}

			// Run plugin preflight hooks (lifecycle stage 3)
			log.Debug("running plugin preflight")
			appCtx := app.PluginContext()
			for _, p := range registry.PreflightsForStartup() {
				if err := p.Preflight(cmd.Context(), appCtx); err != nil {
					return fmt.Errorf("preflight plugin %s: %w", p.Name(), err)
				}
			}

			// Freeze context after initialization — no further mutations allowed
			appCtx.Freeze()

			return nil
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&app.cfgFile, "config", "c", "", "config file (default: .terraci.yaml)")
	rootCmd.PersistentFlags().StringVarP(&app.WorkDir, "dir", "d", cwd, "working directory")
	rootCmd.PersistentFlags().StringVarP(&app.logLevel, "log-level", "l", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output (shorthand for --log-level=debug)")

	// Register core subcommands
	rootCmd.AddCommand(newGenerateCmd(app))
	rootCmd.AddCommand(newGraphCmd(app))
	rootCmd.AddCommand(newValidateCmd(app))
	// Note: summary and policy commands are now provided by plugins
	rootCmd.AddCommand(newInitCmd(app))
	rootCmd.AddCommand(newVersionCmd(app))
	rootCmd.AddCommand(newSchemaCmd())
	rootCmd.AddCommand(newCompletionCmd(rootCmd))
	rootCmd.AddCommand(newManCmd(rootCmd))

	// Register plugin-provided commands
	pluginCtx := app.PluginContext()
	for _, cp := range registry.ByCapability[plugin.CommandProvider]() {
		for _, cmd := range cp.Commands(pluginCtx) {
			rootCmd.AddCommand(cmd)
		}
	}

	return rootCmd
}
