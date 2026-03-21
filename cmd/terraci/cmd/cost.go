package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/internal/cost"
	"github.com/edelwud/terraci/pkg/log"
)

func newCostCmd(app *App) *cobra.Command {
	var (
		costModulePath string
		costOutputFmt  string
	)

	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Estimate AWS costs from Terraform plans",
		Long: `Estimate monthly AWS costs by analyzing plan.json files in module directories.

Scans for plan.json files (output of terraform show -json plan.tfplan),
fetches pricing from the AWS Bulk Pricing API, and calculates cost estimates.

Examples:
  terraci cost                              # Estimate all modules
  terraci cost --module platform/prod/eu-central-1/rds  # Single module
  terraci cost --output json                # JSON output`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if app.Config.Cost == nil || !app.Config.Cost.Enabled {
				log.Error("cost estimation is not enabled (set cost.enabled: true in config)")
				return fmt.Errorf("cost estimation is not enabled")
			}

			log.Info("running cost estimation")

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			estimator := newCostEstimator(app)

			if costModulePath != "" {
				return runCostSingle(ctx, estimator, app, costModulePath, costOutputFmt)
			}
			return runCostAll(ctx, estimator, app, costOutputFmt)
		},
	}

	cmd.Flags().StringVarP(&costModulePath, "module", "m", "", "estimate cost for a specific module")
	cmd.Flags().StringVarP(&costOutputFmt, "output", "o", "text", "output format: text, json")

	return cmd
}

func newCostEstimator(app *App) *cost.Estimator {
	estimator := cost.NewEstimatorFromConfig(app.Config.Cost)

	logFields := log.WithField("dir", estimator.CacheDir()).WithField("ttl", estimator.CacheTTL())
	if age := estimator.CacheOldestAge(); age > 0 {
		remaining := estimator.CacheTTL() - age
		if remaining > 0 {
			logFields = logFields.WithField("expires_in", remaining.Truncate(time.Minute))
		} else {
			logFields = logFields.WithField("status", "expired")
		}
	} else {
		logFields = logFields.WithField("status", "empty")
	}
	logFields.Info("pricing cache")

	return estimator
}

func runCostSingle(ctx context.Context, estimator *cost.Estimator, app *App, costModulePath, costOutputFmt string) error {
	modulePath := filepath.Join(app.WorkDir, costModulePath)
	region := detectRegion(app, costModulePath)

	log.WithField("module", costModulePath).WithField("region", region).Info("estimating module cost")

	mc, err := estimator.EstimateModule(ctx, modulePath, region)
	if err != nil {
		log.WithField("module", costModulePath).WithError(err).Error("estimation failed")
		return fmt.Errorf("estimate module %s: %w", costModulePath, err)
	}

	return outputCostResult(app, costOutputFmt, &cost.EstimateResult{
		Modules:     []cost.ModuleCost{*mc},
		TotalBefore: mc.BeforeCost,
		TotalAfter:  mc.AfterCost,
		TotalDiff:   mc.DiffCost,
		Currency:    "USD",
		GeneratedAt: time.Now().UTC(),
	})
}

func runCostAll(ctx context.Context, estimator *cost.Estimator, app *App, costOutputFmt string) error {
	log.WithField("dir", app.WorkDir).Info("scanning for plan.json files")

	var modulePaths []string
	regions := make(map[string]string)

	err := filepath.Walk(app.WorkDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr // skip walk errors, continue scanning
		}
		if info.Name() == "plan.json" && !info.IsDir() {
			relDir, relErr := filepath.Rel(app.WorkDir, filepath.Dir(path))
			if relErr == nil {
				fullPath := filepath.Dir(path)
				modulePaths = append(modulePaths, fullPath)
				region := detectRegion(app, relDir)
				regions[fullPath] = region
				log.WithField("module", relDir).WithField("region", region).Debug("found plan.json")
			}
		}
		return nil
	})
	if err != nil {
		log.WithError(err).Error("failed to scan for plan.json")
		return fmt.Errorf("scan for plan.json: %w", err)
	}

	if len(modulePaths) == 0 {
		log.Error("no plan.json files found (run terraform plan first)")
		return fmt.Errorf("no plan.json files found")
	}

	log.WithField("count", len(modulePaths)).Info("modules with plan.json found")

	// Prefetch pricing
	log.Info("fetching AWS pricing data")
	if prefetchErr := estimator.ValidateAndPrefetch(ctx, modulePaths, regions); prefetchErr != nil {
		log.WithError(prefetchErr).Warn("failed to prefetch some pricing data")
	}

	log.Info("calculating costs")
	result, err := estimator.EstimateModules(ctx, modulePaths, regions)
	if err != nil {
		log.WithError(err).Error("cost estimation failed")
		return fmt.Errorf("estimate costs: %w", err)
	}

	return outputCostResult(app, costOutputFmt, result)
}

func outputCostResult(app *App, costOutputFmt string, result *cost.EstimateResult) error {
	if costOutputFmt == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Text output
	log.Info("cost estimation results")
	log.IncreasePadding()

	for i := range result.Modules {
		mc := &result.Modules[i]
		status := "✅"
		if mc.Error != "" {
			status = "❌"
		} else if mc.HasChanges {
			status = "🔄"
		}

		moduleID := mc.ModuleID
		if rel, err := filepath.Rel(app.WorkDir, mc.ModulePath); err == nil {
			moduleID = filepath.ToSlash(rel)
		}

		log.WithField("module", moduleID).
			WithField("status", status).
			WithField("monthly", cost.FormatCost(mc.AfterCost)).
			Info("module")

		if mc.HasChanges {
			log.IncreasePadding()
			log.WithField("before", cost.FormatCost(mc.BeforeCost)).
				WithField("after", cost.FormatCost(mc.AfterCost)).
				WithField("diff", cost.FormatCost(mc.DiffCost)).
				Info("cost change")
			log.DecreasePadding()
		}

		if mc.Error != "" {
			log.IncreasePadding()
			log.WithField("error", mc.Error).Warn("estimation error")
			log.DecreasePadding()
		}

		// Show per-resource costs in verbose mode
		if len(mc.Resources) > 0 {
			log.IncreasePadding()
			for _, rc := range mc.Resources {
				if rc.Unsupported {
					continue
				}
				if rc.MonthlyCost > 0 {
					log.WithField("resource", rc.Address).
						WithField("monthly", cost.FormatCost(rc.MonthlyCost)).
						Debug("resource")
				}
			}
			if mc.Unsupported > 0 {
				log.WithField("count", mc.Unsupported).Debug("unsupported resources (usage-based)")
			}
			log.DecreasePadding()
		}
	}

	log.DecreasePadding()

	// Total
	log.Info("total estimated monthly cost")
	log.IncreasePadding()
	if result.TotalDiff != 0 {
		log.WithField("before", cost.FormatCost(result.TotalBefore)).
			WithField("after", cost.FormatCost(result.TotalAfter)).
			WithField("diff", cost.FormatCost(result.TotalDiff)).
			Info("monthly")
	} else {
		log.WithField("monthly", cost.FormatCost(result.TotalAfter)).Info("monthly")
	}
	log.DecreasePadding()

	return nil
}

// detectRegion extracts region from module path using configured pattern segments.
func detectRegion(app *App, modulePath string) string {
	parts := splitPath(modulePath)
	if app.Config.Structure.Segments != nil {
		for i, seg := range app.Config.Structure.Segments {
			if seg == "region" && i < len(parts) {
				return parts[i]
			}
		}
	}
	return "us-east-1"
}

func splitPath(p string) []string {
	var parts []string
	for p != "" && p != "." && p != "/" {
		dir, file := filepath.Split(p)
		if file != "" {
			parts = append([]string{file}, parts...)
		}
		p = filepath.Clean(dir)
	}
	return parts
}
