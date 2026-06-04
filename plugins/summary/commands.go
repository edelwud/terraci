package summary

import (
	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/plugin"
)

// CommandSpecs returns the `terraci summary` command.
func (p *Plugin) CommandSpecs() ([]plugin.CommandSpec, error) {
	cmd, err := plugin.NewCommandSpec(plugin.CommandSpecOptions{
		Use:   pluginName,
		Short: "Create MR/PR comment from plan results",
		Long: `Collects terraform plan results from artifacts and creates/updates
a summary comment on the merge/pull request.

This command is designed to run as a final job in the pipeline after all
plan jobs have completed. It scans for plan results in module directories
and posts a formatted comment to the MR/PR.

Example:
  terraci summary`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmdCtx, current, err := plugin.CommandPlugin[*Plugin](cmd, p.Name())
			if err != nil {
				return err
			}
			return current.runSummary(cmd.Context(), cmdCtx.AppContext())
		},
	})
	if err != nil {
		return nil, err
	}
	return []plugin.CommandSpec{cmd}, nil
}
