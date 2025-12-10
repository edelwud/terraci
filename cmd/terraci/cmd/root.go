package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
)

var (
	// Global flags
	cfgFile  string
	workDir  string
	logLevel string

	// Version info
	versionInfo struct {
		Version string
		Commit  string
		Date    string
	}

	// Global config
	cfg *config.Config
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "terraci",
	Short: "Generate CI pipelines for Terraform projects",
	Long: `TerraCi is a CLI tool that analyzes Terraform project structure,
builds a dependency graph based on terraform_remote_state references,
and generates CI pipelines (GitLab CI) that respect those dependencies.

Features:
  - Automatic discovery of Terraform modules
  - Dependency graph from terraform_remote_state
  - Support for for_each in remote state
  - Glob pattern filtering for modules
  - Git integration for changed-only pipelines
  - Parallel execution where possible`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Initialize logger
		log.Init()

		// Handle verbose flag (shorthand for --log-level=debug)
		if verbose, err := cmd.Flags().GetBool("verbose"); err == nil && verbose {
			logLevel = "debug"
		}

		// Set log level from flag
		if logLevel != "" {
			if err := log.SetLevelFromString(logLevel); err != nil {
				return fmt.Errorf("invalid log level %q: %w", logLevel, err)
			}
		}

		// Show version info (skip for version command itself)
		if cmd.Name() != "version" && versionInfo.Version != "" {
			log.WithField("version", versionInfo.Version).Debug("terraci")
		}

		// Skip config loading for version, schema, and completion commands
		if cmd.Name() == "version" || cmd.Name() == "schema" || cmd.Name() == "completion" {
			return nil
		}

		// Load configuration
		log.Debug("loading configuration")
		var err error
		if cfgFile != "" {
			log.WithField("file", cfgFile).Debug("loading config from file")
			cfg, err = config.Load(cfgFile)
		} else {
			log.WithField("dir", workDir).Debug("loading config from directory")
			cfg, err = config.LoadOrDefault(workDir)
		}

		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		log.Debug("validating configuration")
		return cfg.Validate()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets version information
func SetVersion(version, commit, date string) {
	versionInfo.Version = version
	versionInfo.Commit = commit
	versionInfo.Date = date
}

func init() {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: .terraci.yaml)")
	rootCmd.PersistentFlags().StringVarP(&workDir, "dir", "d", cwd, "working directory")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output (shorthand for --log-level=debug)")
}
