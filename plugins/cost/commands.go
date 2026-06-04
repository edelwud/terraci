package cost

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin"
)

const (
	defaultEstimationTimeout = 5 * time.Minute
	defaultOutputFormat      = "text"
)

// CommandSpecs returns the CLI commands provided by the cost plugin.
func (p *Plugin) CommandSpecs() ([]plugin.CommandSpec, error) {
	var (
		costModulePath string
		costOutputFmt  string
	)

	cmd, err := plugin.NewCommandSpec(plugin.CommandSpecOptions{
		Use:   pluginName,
		Short: "Estimate cloud costs from Terraform plans",
		Long: `Estimate monthly cloud costs by analyzing plan.json files in module directories.

Examples:
  terraci cost
  terraci cost --module platform/prod/eu-central-1/rds
  terraci cost --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmdCtx, current, err := plugin.CommandPlugin[*Plugin](cmd, p.Name())
			if err != nil {
				return err
			}
			if err := plugin.RequireEnabled(current, "cost estimation is not enabled (enable at least one provider under extensions.cost.providers)"); err != nil {
				return err
			}

			log.Info("cost: running cost estimation")
			c, cancel := context.WithTimeout(cmd.Context(), defaultEstimationTimeout)
			defer cancel()

			return current.runEstimation(c, cmdCtx.AppContext(), costModulePath, costOutputFmt)
		},
		Configure: func(cmd *cobra.Command) error {
			cmd.Flags().StringVarP(&costModulePath, "module", "m", "", "estimate cost for a specific module")
			cmd.Flags().StringVarP(&costOutputFmt, "output", "o", defaultOutputFormat, "output format: text, json")
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	return []plugin.CommandSpec{cmd}, nil
}
