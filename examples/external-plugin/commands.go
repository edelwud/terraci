package hello

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/workflow"
)

// CommandSpecs returns the `terraci hello` command.
func (p *Plugin) CommandSpecs() ([]plugin.CommandSpec, error) {
	cmd, err := plugin.NewCommandSpec(plugin.CommandSpecOptions{
		Use:   "hello",
		Short: "Print discovered Terraform modules",
		Long: `Example external plugin command. Scans for Terraform modules
using the configured structure pattern and prints a summary.

Build a custom binary with this plugin:
  xterraci build --with github.com/edelwud/terraci/examples/external-plugin=./examples/external-plugin`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			appCtx, current, err := plugin.CommandPlugin[*Plugin](cmd, p.Name())
			if err != nil {
				return err
			}
			return runHello(cmd.Context(), appCtx, current.greeting())
		},
	})
	if err != nil {
		return nil, err
	}
	return []plugin.CommandSpec{cmd}, nil
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

	project, err := workflow.PlanProject(ctx, workflow.ProjectRequest{
		WorkDir: appCtx.WorkDir(),
		Config:  appCtx.Config(),
	})
	if err != nil {
		return fmt.Errorf("plan project: %w", err)
	}
	modules := project.Workflow.Filtered.All()
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
