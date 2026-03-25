package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func newManCmd(rootCmd *cobra.Command) *cobra.Command {
	var manDir string

	cmd := &cobra.Command{
		Use:    "man",
		Short:  "Generate man pages",
		Hidden: true,
		Long: `Generate man pages for xterraci.

Example:
  xterraci man -d ./man
  sudo cp ./man/*.1 /usr/share/man/man1/
`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if manDir == "" {
				manDir = "man"
			}

			if err := os.MkdirAll(manDir, 0o755); err != nil {
				return fmt.Errorf("failed to create man directory: %w", err)
			}

			now := time.Now()
			header := &doc.GenManHeader{
				Title:   "XTERRACI",
				Section: "1",
				Date:    &now,
				Source:  "xterraci",
				Manual:  "xterraci Manual",
			}

			if err := doc.GenManTree(rootCmd, header, manDir); err != nil {
				return fmt.Errorf("failed to generate man pages: %w", err)
			}

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

	cmd.Flags().StringVarP(&manDir, "dir", "d", "man", "output directory for man pages")

	return cmd
}
