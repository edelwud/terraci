package cost

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
)

// Commands returns the CLI commands provided by the cost plugin.
func (p *Plugin) Commands(ctx *plugin.AppContext) []*cobra.Command {
	var (
		costModulePath string
		costOutputFmt  string
	)

	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Estimate cloud costs from Terraform plans",
		Long: `Estimate monthly cloud costs by analyzing plan.json files in module directories.

Examples:
  terraci cost
  terraci cost --module platform/prod/eu-central-1/rds
  terraci cost --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !p.IsEnabled() {
				return errors.New("cost estimation is not enabled (enable at least one provider under plugins.cost.providers)")
			}

			log.Info("running cost estimation")
			c, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			return p.runEstimation(c, ctx, costModulePath, costOutputFmt)
		},
	}

	cmd.Flags().StringVarP(&costModulePath, "module", "m", "", "estimate cost for a specific module")
	cmd.Flags().StringVarP(&costOutputFmt, "output", "o", "text", "output format: text, json")

	return []*cobra.Command{cmd}
}
