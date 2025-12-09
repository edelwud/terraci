package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
)

var (
	forceInit bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize terraci configuration",
	Long: `Create a default .terraci.yaml configuration file in the current directory.

This will create a configuration file with sensible defaults that you can
customize for your project.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "overwrite existing config file")
}

func runInit(_ *cobra.Command, _ []string) error {
	configPath := filepath.Join(workDir, ".terraci.yaml")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil && !forceInit {
		return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
	}

	// Create default config
	log.Debug("creating default configuration")
	defaultCfg := config.DefaultConfig()

	// Write config
	if err := defaultCfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	log.WithField("file", configPath).Info("configuration created")
	log.Info("you can now customize the configuration and run:")
	log.IncreasePadding()
	log.Info("terraci generate -o .gitlab-ci.yml")
	log.DecreasePadding()

	return nil
}
