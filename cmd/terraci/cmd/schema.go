package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func newSchemaCmd(app *App) *cobra.Command {
	var schemaOutputFile string

	cmd := &cobra.Command{
		Use:         "schema",
		Short:       "Generate JSON Schema for .terraci.yaml",
		Annotations: map[string]string{"skipConfig": "true"},
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
		RunE: func(_ *cobra.Command, _ []string) error {
			pluginSchemas := make(map[string]any)
			for _, cl := range registry.ByCapabilityFrom[plugin.ConfigLoader](app.Plugins) {
				pluginSchemas[cl.ConfigKey()] = cl.NewConfig()
			}
			schema := config.GenerateJSONSchema(pluginSchemas)

			if schemaOutputFile != "" {
				if err := os.WriteFile(schemaOutputFile, []byte(schema), 0o600); err != nil {
					return fmt.Errorf("failed to write schema file: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Schema written to %s\n", schemaOutputFile)
			} else {
				fmt.Print(schema)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&schemaOutputFile, "output", "o", "", "output file (default: stdout)")

	return cmd
}
