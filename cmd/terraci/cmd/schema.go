package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
)

var (
	schemaOutputFile string
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Generate JSON Schema for .terraci.yaml",
	Long: `Generate a JSON Schema file for .terraci.yaml configuration.

The schema can be used for IDE autocompletion and validation.

Examples:
  # Output schema to stdout
  terraci schema

  # Write schema to file
  terraci schema -o terraci.schema.json

  # Use in VS Code with YAML extension
  # Add to .terraci.yaml:
  # yaml-language-server: $schema=./terraci.schema.json`,
	RunE: runSchema,
}

func init() {
	rootCmd.AddCommand(schemaCmd)

	schemaCmd.Flags().StringVarP(&schemaOutputFile, "output", "o", "", "output file (default: stdout)")
}

func runSchema(_ *cobra.Command, _ []string) error {
	schema := config.GenerateJSONSchema()

	if schemaOutputFile != "" {
		if err := os.WriteFile(schemaOutputFile, []byte(schema), 0o600); err != nil {
			return fmt.Errorf("failed to write schema file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Schema written to %s\n", schemaOutputFile)
	} else {
		fmt.Print(schema)
	}

	return nil
}
