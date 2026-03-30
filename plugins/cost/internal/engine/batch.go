package engine

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

type estimateCoordinator struct {
	scanner    *ModuleScanner
	planner    *PrefetchPlanner
	executor   *ModuleExecutor
	prefetcher interface {
		PrefetchPricing(context.Context, costruntime.ServicePlan) error
	}
	providerMetadata func() map[string]model.ProviderMetadata
}

func newEstimateCoordinator(
	scanner *ModuleScanner,
	planner *PrefetchPlanner,
	executor *ModuleExecutor,
	prefetcher interface {
		PrefetchPricing(context.Context, costruntime.ServicePlan) error
	},
	providerMetadata func() map[string]model.ProviderMetadata,
) *estimateCoordinator {
	return &estimateCoordinator{
		scanner:          scanner,
		planner:          planner,
		executor:         executor,
		prefetcher:       prefetcher,
		providerMetadata: providerMetadata,
	}
}

// Estimate estimates multiple modules with best-effort scan semantics.
func (b *estimateCoordinator) Estimate(ctx context.Context, modulePaths []string, regions map[string]string) (*model.EstimateResult, error) {
	const maxConcurrency = 4

	results := make([]model.ModuleCost, len(modulePaths))
	scannedPlans := b.scanner.ScanManyBestEffort(modulePaths, regions)

	executablePlans := make([]ScannedModulePlan, 0, len(scannedPlans))
	modulePlans := make([]*ModulePlan, 0, len(scannedPlans))
	for _, scanned := range scannedPlans {
		if scanned.Err != nil {
			results[scanned.Index] = moduleCostFromScanError(scanned.ModulePath, scanned.Region, scanned.Err)
			continue
		}

		executablePlans = append(executablePlans, scanned)
		modulePlans = append(modulePlans, scanned.Plan)
	}

	if prefetchErr := b.prefetcher.PrefetchPricing(ctx, b.planner.Build(modulePlans)); prefetchErr != nil {
		log.WithError(prefetchErr).Warn("failed to prefetch some pricing data")
	}

	var g errgroup.Group
	g.SetLimit(maxConcurrency)
	for _, scanned := range executablePlans {
		g.Go(func() error {
			results[scanned.Index] = *b.executor.Execute(ctx, scanned.Plan)
			return nil
		})
	}
	_ = g.Wait() //nolint:errcheck // individual errors collected in results

	result := &model.EstimateResult{
		Modules:          results,
		Currency:         "USD",
		GeneratedAt:      time.Now().UTC(),
		ProviderMetadata: b.providerMetadata(),
	}
	providerSet := make(map[string]bool)
	for i := range results {
		result.TotalBefore += results[i].BeforeCost
		result.TotalAfter += results[i].AfterCost
		for _, providerID := range results[i].Providers {
			providerSet[providerID] = true
		}
		if results[i].Error != "" {
			result.Errors = append(result.Errors, model.ModuleError{
				ModuleID: results[i].ModuleID,
				Error:    results[i].Error,
			})
		}
	}
	result.TotalDiff = result.TotalAfter - result.TotalBefore
	for providerID := range providerSet {
		result.Providers = append(result.Providers, providerID)
	}

	return result, nil
}

func moduleCostFromScanError(modulePath, region string, err error) model.ModuleCost {
	return model.ModuleCost{
		ModuleID:   strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath: modulePath,
		Region:     region,
		Error:      err.Error(),
	}
}
