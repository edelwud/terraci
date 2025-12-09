package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
)

var (
	// Global flags
	cfgFile string
	workDir string
	verbose bool

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
		// Skip config loading for version command
		if cmd.Name() == "version" {
			return nil
		}

		// Load configuration
		var err error
		if cfgFile != "" {
			cfg, err = config.Load(cfgFile)
		} else {
			cfg, err = config.LoadOrDefault(workDir)
		}

		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

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
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
