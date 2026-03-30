package summary

import (
	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/plugin"
)

// Commands returns the `terraci summary` command.
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
	return []*cobra.Command{{
		Use:   "summary",
		Short: "Create MR/PR comment from plan results",
		Long: `Collects terraform plan results from artifacts and creates/updates
a summary comment on the merge/pull request.

This command is designed to run as a final job in the pipeline after all
plan jobs have completed. It scans for plan results in module directories
and posts a formatted comment to the MR/PR.

Example:
  terraci summary`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return p.runSummary(cmd.Context(), ctx)
		},
	}}
}
