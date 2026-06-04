package skeleton

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/plugin"
)

// CommandSpecs implements plugin.CommandProvider — registers `terraci skeleton`.
//
// CommandPlugin is the canonical callback boundary: it returns both the
// command context and the command-scoped plugin instance. The framework
// rebuilds the registry for every command run, so state captured at command
// registration time would be stale.
func (p *Plugin) CommandSpecs() ([]plugin.CommandSpec, error) {
	var consumeMode bool

	cmd, err := plugin.NewCommandSpec(plugin.CommandSpecOptions{
		Use:   pluginName,
		Short: "Skeleton plugin — demonstrates producer + consumer patterns",
		Long: `Skeleton plugin command. Without flags, runs the producer flow:
collects a tiny report payload and writes skeleton-report.json into the
service directory. With --consume, runs the consumer flow: loads every
*-report.json (except its own) and prints a brief summary.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmdCtx, current, err := plugin.CommandPlugin[*Plugin](cmd, pluginName)
			if err != nil {
				return err
			}
			if err := plugin.RequireEnabled(current, "skeleton plugin is not enabled — set extensions.skeleton.enabled: true"); err != nil {
				return err
			}

			runtime := NewRuntime(cmdCtx.AppContext(), current.Config())
			result, err := Run(cmd.Context(), runtime, Request{Consume: consumeMode})
			if err != nil {
				return err
			}
			return WriteOutput(os.Stdout, result)
		},
		Configure: func(cmd *cobra.Command) error {
			cmd.Flags().BoolVar(&consumeMode, "consume", false, "read other plugins' *-report.json instead of writing one")
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return []plugin.CommandSpec{cmd}, nil
}
