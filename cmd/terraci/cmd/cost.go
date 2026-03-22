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
	for i := range result.Modules {
		mc := &result.Modules[i]
		if mc.AfterCost == 0 && mc.BeforeCost == 0 && mc.Error == "" {
			continue
		}

		moduleID := mc.ModuleID
		if rel, err := filepath.Rel(app.WorkDir, mc.ModulePath); err == nil {
			moduleID = filepath.ToSlash(rel)
		}

		moduleEntry := log.WithField("monthly", cost.FormatCost(mc.AfterCost))
		if mc.HasChanges {
			moduleEntry = moduleEntry.WithField("diff", cost.FormatCostDiff(mc.DiffCost))
		}
		if mc.Error != "" {
			moduleEntry = moduleEntry.WithField("error", mc.Error)
		}
		moduleEntry.Info(moduleID)

		log.IncreasePadding()
		for j := range mc.Submodules {
			sm := &mc.Submodules[j]
			if sm.MonthlyCost == 0 {
				continue
			}

			// Show submodule header if there's more than one group
			showGroup := len(mc.Submodules) > 1
			if showGroup {
				label := sm.ModuleAddr
				if label == "" {
					label = "(root)"
				}
				log.WithField("monthly", cost.FormatCost(sm.MonthlyCost)).Info(label)
				log.IncreasePadding()
			}

			for k := range sm.Resources {
				rc := &sm.Resources[k]
				// Strip module prefix from address for cleaner output
				displayAddr := stripModulePrefix(rc.Address, sm.ModuleAddr)

				switch rc.ErrorKind {
				case cost.CostErrorNone:
					if rc.MonthlyCost > 0 {
						entry := log.WithField("monthly", cost.FormatCost(rc.MonthlyCost))
						for dk, dv := range rc.Details {
							entry = entry.WithField(dk, dv)
						}
						entry.Info(displayAddr)
					}
				case cost.CostErrorUsageBased:
					log.WithField("note", "usage-based").Debug(displayAddr)
				case cost.CostErrorNoHandler:
					log.WithField("note", "unsupported").Debug(displayAddr)
				case cost.CostErrorLookupFailed, cost.CostErrorAPIFailure, cost.CostErrorNoPrice:
					log.WithField("error", rc.ErrorDetail).Warn(displayAddr)
				}
			}

			if showGroup {
				log.DecreasePadding()
			}
		}
		log.DecreasePadding()
	}

	// Total
	if result.TotalDiff != 0 {
		log.WithField("before", cost.FormatCost(result.TotalBefore)).
			WithField("after", cost.FormatCost(result.TotalAfter)).
			WithField("diff", cost.FormatCostDiff(result.TotalDiff)).
			Info("total")
	} else {
		log.WithField("monthly", cost.FormatCost(result.TotalAfter)).Info("total")
	}

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

// stripModulePrefix removes the "module.x.module.y." prefix from a resource address
// when displayed inside its module group, since it's redundant.
// e.g., "module.runner.aws_instance.web" with prefix "module.runner" → "aws_instance.web"
func stripModulePrefix(address, moduleAddr string) string {
	if moduleAddr == "" {
		return address
	}
	prefix := moduleAddr + "."
	if len(address) > len(prefix) && address[:len(prefix)] == prefix {
		return address[len(prefix):]
	}
	return address
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
