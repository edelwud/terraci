package skeleton

import (
	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/plugin"
)

// Commands implements plugin.CommandProvider — registers `terraci skeleton`.
//
// CommandPlugin is the canonical callback boundary: it returns both the
// per-run AppContext and the command-scoped plugin instance. The framework
// rebuilds the registry for every command run, so state captured at command
// registration time would be stale.
func (p *Plugin) Commands() []*cobra.Command {
	var consumeMode bool

	cmd := &cobra.Command{
		Use:   pluginName,
		Short: "Skeleton plugin — demonstrates producer + consumer patterns",
		Long: `Skeleton plugin command. Without flags, runs the producer flow:
collects a tiny report payload and writes skeleton-report.json into the
service directory. With --consume, runs the consumer flow: loads every
*-report.json (except its own) and prints a brief summary.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			appCtx, current, err := plugin.CommandPlugin[*Plugin](cmd, pluginName)
			if err != nil {
				return err
			}
			if err := plugin.RequireEnabled(current, "skeleton plugin is not enabled — set extensions.skeleton.enabled: true"); err != nil {
				return err
			}

			if consumeMode {
				return runConsumer(cmd.Context(), appCtx)
			}
			return runProducer(cmd.Context(), appCtx, current.Config())
		},
	}

	cmd.Flags().BoolVar(&consumeMode, "consume", false, "read other plugins' *-report.json instead of writing one")
	return []*cobra.Command{cmd}
}
