package cost

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
)

const (
	defaultEstimationTimeout = 5 * time.Minute
	defaultOutputFormat      = "text"
)

// Commands returns the CLI commands provided by the cost plugin.
func (p *Plugin) Commands(appCtx *plugin.AppContext) []*cobra.Command {
	var (
		costModulePath string
		costOutputFmt  string
	)

	cmd := &cobra.Command{
		Use:   pluginName,
		Short: "Estimate cloud costs from Terraform plans",
		Long: `Estimate monthly cloud costs by analyzing plan.json files in module directories.

Examples:
  terraci cost
  terraci cost --module platform/prod/eu-central-1/rds
  terraci cost --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			current, err := plugin.CommandInstance[*Plugin](appCtx, p.Name())
			if err != nil {
				return err
			}
			if !current.IsEnabled() {
				return errors.New("cost estimation is not enabled (enable at least one provider under plugins.cost.providers)")
			}

			log.Info("cost: running cost estimation")
			c, cancel := context.WithTimeout(cmd.Context(), defaultEstimationTimeout)
			defer cancel()

			return current.runEstimation(c, appCtx, costModulePath, costOutputFmt)
		},
	}

	cmd.Flags().StringVarP(&costModulePath, "module", "m", "", "estimate cost for a specific module")
	cmd.Flags().StringVarP(&costOutputFmt, "output", "o", defaultOutputFormat, "output format: text, json")

	return []*cobra.Command{cmd}
}
