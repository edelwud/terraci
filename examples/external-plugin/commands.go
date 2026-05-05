package hello

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/plugin"
)

// Commands returns the `terraci hello` command.
func (p *Plugin) Commands() []*cobra.Command {
	return []*cobra.Command{{
		Use:   "hello",
		Short: "Print discovered Terraform modules",
		Long: `Example external plugin command. Scans for Terraform modules
using the configured structure pattern and prints a summary.

Build a custom binary with this plugin:
  xterraci build --with github.com/edelwud/terraci/examples/external-plugin=./examples/external-plugin`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			appCtx := plugin.FromContext(cmd.Context())
			current, err := plugin.CommandInstance[*Plugin](appCtx, p.Name())
			if err != nil {
				return err
			}
			return runHello(cmd.Context(), appCtx, current.greeting())
		},
	}}
}

func (p *Plugin) greeting() string {
	if cfg := p.Config(); cfg != nil && cfg.Greeting != "" {
		return cfg.Greeting
	}
	return "Hello from TerraCi plugin!"
}

func runHello(ctx context.Context, appCtx *plugin.AppContext, greeting string) error {
	fmt.Println(greeting)
	fmt.Println()

	cfg := appCtx.Config()
	segments, err := config.ParsePattern(cfg.Structure.Pattern)
	if err != nil {
		return fmt.Errorf("parse pattern: %w", err)
	}

	scanner := discovery.NewScanner(appCtx.WorkDir(), segments)
	modules, err := scanner.Scan(ctx)
	if err != nil {
		return fmt.Errorf("scan modules: %w", err)
	}

	if len(modules) == 0 {
		fmt.Println("No Terraform modules found.")
		return nil
	}

	fmt.Printf("Found %d module(s):\n\n", len(modules))
	for _, m := range modules {
		fmt.Printf("  - %s\n", m.Path)
	}

	return nil
}
