package engine

import (
	"context"
	"time"

	"github.com/caarlos0/log"
	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/results"
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

	moduleResults := make([]model.ModuleCost, len(modulePaths))
	scannedPlans := b.scanner.ScanManyBestEffort(modulePaths, regions)

	executablePlans := make([]ScannedModulePlan, 0, len(scannedPlans))
	modulePlans := make([]*ModulePlan, 0, len(scannedPlans))
	for _, scanned := range scannedPlans {
		if scanned.Err != nil {
			moduleResults[scanned.Index] = results.NewErroredModule(scanned.ModulePath, scanned.Region, scanned.Err)
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
			moduleResults[scanned.Index] = *b.executor.Execute(ctx, scanned.Plan)
			return nil
		})
	}
	_ = g.Wait() //nolint:errcheck // individual errors collected in results

	assembler := results.NewEstimateAssembler(b.providerMetadata(), time.Now().UTC())
	for i := range moduleResults {
		assembler.AddModule(moduleResults[i])
	}

	return assembler.Build(), nil
}
