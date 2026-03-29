package cost

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	costengine "github.com/edelwud/terraci/plugins/cost/internal"
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
				return errors.New("cost estimation is not enabled (set plugins.cost.providers.aws.enabled: true)")
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

func (p *Plugin) runEstimation(ctx context.Context, appCtx *plugin.AppContext, modulePath, outputFmt string) error {
	cfg := appCtx.Config()
	workDir := appCtx.WorkDir()
	serviceDir := appCtx.ServiceDir()

	log.WithField("dir", workDir).Info("scanning for plan.json files")

	modulePaths, err := discovery.FindModulesWithPlan(workDir)
	if err != nil {
		return fmt.Errorf("scan for plan.json: %w", err)
	}

	if modulePath != "" {
		target := filepath.Join(workDir, modulePath)
		filtered := make([]string, 0, 1)
		for _, p := range modulePaths {
			if p == target || strings.HasSuffix(p, modulePath) {
				filtered = append(filtered, p)
			}
		}
		modulePaths = filtered
	}

	if len(modulePaths) == 0 {
		return errors.New("no plan.json files found")
	}

	log.WithField("count", len(modulePaths)).Info("modules with plan.json found")

	regions := make(map[string]string)
	for _, fullPath := range modulePaths {
		relDir, relErr := filepath.Rel(workDir, fullPath)
		if relErr == nil {
			regions[fullPath] = costengine.DetectRegion(cfg.Structure.Segments, relDir)
		}
	}

	estimator := p.getEstimator()
	if estimator == nil {
		return errors.New("cost estimator not initialized (check plugins.cost configuration)")
	}

	result, err := estimator.EstimateModules(ctx, modulePaths, regions)
	if err != nil {
		return fmt.Errorf("estimate costs: %w", err)
	}

	if serviceDir != "" {
		if saveErr := ci.SaveJSON(serviceDir, resultsFile, result); saveErr != nil {
			log.WithError(saveErr).Warn("failed to save cost results")
		}
		report := buildCostReport(result)
		if saveErr := ci.SaveReport(serviceDir, report); saveErr != nil {
			log.WithError(saveErr).Warn("failed to save cost report")
		}
	}

	return p.outputResult(os.Stdout, appCtx, outputFmt, result)
}

func buildCostReport(result *costengine.EstimateResult) *ci.Report {
	modules := make([]ci.ModuleReport, 0, len(result.Modules))
	for i := range result.Modules {
		if result.Modules[i].Error == "" {
			modules = append(modules, ci.ModuleReport{
				ModulePath: result.Modules[i].ModulePath,
				CostBefore: result.Modules[i].BeforeCost,
				CostAfter:  result.Modules[i].AfterCost,
				CostDiff:   result.Modules[i].DiffCost,
				HasCost:    true,
			})
		}
	}

	return &ci.Report{
		Plugin:  "cost",
		Title:   "Cost Estimation",
		Status:  ci.ReportStatusPass,
		Summary: fmt.Sprintf("%d modules, total: $%.2f/mo (diff: %+.2f)", len(result.Modules), result.TotalAfter, result.TotalDiff),
		Body:    renderCostReportBody(result),
		Modules: modules,
	}
}

func renderCostReportBody(result *costengine.EstimateResult) string {
	var b strings.Builder
	b.WriteString("| Module | Before | After | Diff |\n")
	b.WriteString("|--------|--------|-------|------|\n")
	for i := range result.Modules {
		if result.Modules[i].Error != "" {
			continue
		}
		fmt.Fprintf(&b, "| %s | $%.2f | $%.2f | %+.2f |\n",
			result.Modules[i].ModulePath, result.Modules[i].BeforeCost, result.Modules[i].AfterCost, result.Modules[i].DiffCost)
	}
	fmt.Fprintf(&b, "\n**Total:** $%.2f/mo (diff: %+.2f)\n", result.TotalAfter, result.TotalDiff)
	return b.String()
}
