package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/ci"
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
		Short: "Estimate AWS costs from Terraform plans",
		Long: `Estimate monthly AWS costs by analyzing plan.json files in module directories.

Examples:
  terraci cost
  terraci cost --module platform/prod/eu-central-1/rds
  terraci cost --output json`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if !p.IsConfigured() {
				return fmt.Errorf("cost estimation is not enabled (set plugins.cost.enabled: true)")
			}
			if !p.cfg.Enabled {
				return fmt.Errorf("cost estimation is not enabled (set plugins.cost.enabled: true)")
			}

			log.Info("running cost estimation")
			c, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			estimator := p.getEstimator()

			if costModulePath != "" {
				return p.runSingle(c, estimator, ctx, costModulePath, costOutputFmt)
			}
			return p.runAll(c, estimator, ctx, costOutputFmt)
		},
	}

	cmd.Flags().StringVarP(&costModulePath, "module", "m", "", "estimate cost for a specific module")
	cmd.Flags().StringVarP(&costOutputFmt, "output", "o", "text", "output format: text, json")

	return []*cobra.Command{cmd}
}

func (p *Plugin) runSingle(ctx context.Context, estimator *costengine.Estimator, appCtx *plugin.AppContext, modulePath, outputFmt string) error {
	fullPath := filepath.Join(appCtx.WorkDir, modulePath)
	region := costengine.DetectRegion(appCtx.Config.Structure.Segments, modulePath)

	log.WithField("module", modulePath).WithField("region", region).Info("estimating module cost")

	mc, err := estimator.EstimateModule(ctx, fullPath, region)
	if err != nil {
		return fmt.Errorf("estimate module %s: %w", modulePath, err)
	}

	result := &costengine.EstimateResult{
		Modules:     []costengine.ModuleCost{*mc},
		TotalBefore: mc.BeforeCost,
		TotalAfter:  mc.AfterCost,
		TotalDiff:   mc.DiffCost,
		Currency:    "USD",
		GeneratedAt: time.Now().UTC(),
	}

	if p.serviceDir != "" {
		if saveErr := saveCostResults(p.serviceDir, result); saveErr != nil {
			log.WithError(saveErr).Warn("failed to save cost results")
		}
		report := buildCostReport(result)
		if saveErr := saveReport(p.serviceDir, report); saveErr != nil {
			log.WithError(saveErr).Warn("failed to save cost report")
		}
	}

	return p.outputResult(appCtx, outputFmt, result)
}

func (p *Plugin) runAll(ctx context.Context, estimator *costengine.Estimator, appCtx *plugin.AppContext, outputFmt string) error {
	log.WithField("dir", appCtx.WorkDir).Info("scanning for plan.json files")

	var modulePaths []string
	regions := make(map[string]string)

	err := filepath.Walk(appCtx.WorkDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.Name() == "plan.json" && !info.IsDir() {
			relDir, relErr := filepath.Rel(appCtx.WorkDir, filepath.Dir(path))
			if relErr == nil {
				fullPath := filepath.Dir(path)
				modulePaths = append(modulePaths, fullPath)
				region := costengine.DetectRegion(appCtx.Config.Structure.Segments, relDir)
				regions[fullPath] = region
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scan for plan.json: %w", err)
	}

	if len(modulePaths) == 0 {
		return fmt.Errorf("no plan.json files found")
	}

	log.WithField("count", len(modulePaths)).Info("modules with plan.json found")

	if prefetchErr := estimator.ValidateAndPrefetch(ctx, modulePaths, regions); prefetchErr != nil {
		log.WithError(prefetchErr).Warn("failed to prefetch some pricing data")
	}

	result, err := estimator.EstimateModules(ctx, modulePaths, regions)
	if err != nil {
		return fmt.Errorf("estimate costs: %w", err)
	}

	if p.serviceDir != "" {
		if saveErr := saveCostResults(p.serviceDir, result); saveErr != nil {
			log.WithError(saveErr).Warn("failed to save cost results")
		}
		report := buildCostReport(result)
		if saveErr := saveReport(p.serviceDir, report); saveErr != nil {
			log.WithError(saveErr).Warn("failed to save cost report")
		}
	}

	return p.outputResult(appCtx, outputFmt, result)
}

// saveCostResults writes the cost estimation result to the service directory as JSON.
func saveCostResults(serviceDir string, result *costengine.EstimateResult) error {
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		return fmt.Errorf("create service dir: %w", err)
	}
	path := filepath.Join(serviceDir, "cost-results.json")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create cost results file: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
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

func saveReport(serviceDir string, report *ci.Report) error {
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(serviceDir, report.Plugin+"-report.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
