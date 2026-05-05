package skeleton

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/plugin"
)

// Commands implements plugin.CommandProvider — registers `terraci skeleton`.
//
// CommandInstance is the canonical way to fetch the per-command plugin
// instance: the framework rebuilds the registry for every command run, so
// any state captured at command-registration time would be stale.
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
			appCtx := plugin.FromContext(cmd.Context())
			current, err := plugin.CommandInstance[*Plugin](appCtx, pluginName)
			if err != nil {
				return err
			}
			if !current.IsEnabled() {
				return fmt.Errorf("skeleton plugin is not enabled — set extensions.skeleton.enabled: true")
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
