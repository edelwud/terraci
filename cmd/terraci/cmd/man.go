package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var manDir string

var manCmd = &cobra.Command{
	Use:    "man",
	Short:  "Generate man pages",
	Hidden: true,
	Long: `Generate man pages for terraci.

This command generates man pages in roff format that can be installed
to /usr/share/man/man1/ or similar directories.

Example:
  # Generate man pages to ./man directory
  terraci man -d ./man

  # Install man pages (Linux)
  sudo cp ./man/*.1 /usr/share/man/man1/
  sudo mandb

  # View man page
  man terraci
`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if manDir == "" {
			manDir = "man"
		}

		// Create output directory
		if err := os.MkdirAll(manDir, 0o755); err != nil {
			return fmt.Errorf("failed to create man directory: %w", err)
		}

		// Generate man pages with header
		header := &doc.GenManHeader{
			Title:   "TERRACI",
			Section: "1",
			Date:    &time.Time{},
			Source:  "TerraCi",
			Manual:  "TerraCi Manual",
		}

		if err := doc.GenManTree(rootCmd, header, manDir); err != nil {
			return fmt.Errorf("failed to generate man pages: %w", err)
		}

		// List generated files
		files, err := filepath.Glob(filepath.Join(manDir, "*.1"))
		if err != nil {
			return err
		}

		fmt.Printf("Generated %d man pages in %s:\n", len(files), manDir)
		for _, f := range files {
			fmt.Printf("  - %s\n", filepath.Base(f))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(manCmd)

	manCmd.Flags().StringVarP(&manDir, "dir", "d", "man", "output directory for man pages")
}
