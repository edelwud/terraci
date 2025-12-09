package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
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
	defaultCfg := config.DefaultConfig()

	// Write config
	if err := defaultCfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	fmt.Printf("Created %s\n", configPath)
	fmt.Println("\nYou can now customize the configuration and run:")
	fmt.Println("  terraci generate -o .gitlab-ci.yml")

	return nil
}
