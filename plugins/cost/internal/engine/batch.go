package engine

import (
	"context"
	"sync"
	"time"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/results"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

// indexWarmer downloads pricing indexes for a set of service/region pairs
// before estimation begins, reducing per-resource latency during concurrent execution.
type indexWarmer interface {
	WarmIndexes(ctx context.Context, services map[pricing.ServiceID][]string) error
}

type estimateCoordinator struct {
	scanner          *ModuleScanner
	executor         *ModuleExecutor
	providerMetadata func() map[string]model.ProviderMetadata
	catalog          costruntime.ProviderCatalogRuntime
	runtimes         indexWarmer
}

func newEstimateCoordinator(
	scanner *ModuleScanner,
	executor *ModuleExecutor,
	catalog costruntime.ProviderCatalogRuntime,
	providerMetadata func() map[string]model.ProviderMetadata,
	runtimes indexWarmer,
) *estimateCoordinator {
	return &estimateCoordinator{
		scanner:          scanner,
		executor:         executor,
		catalog:          catalog,
		providerMetadata: providerMetadata,
		runtimes:         runtimes,
	}
}

// Estimate estimates multiple modules with best-effort scan semantics.
func (b *estimateCoordinator) Estimate(ctx context.Context, modulePaths []string, regions map[string]string) (*model.EstimateResult, error) {
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

	if b.runtimes != nil && b.catalog != nil {
		requirements := buildPrefetchRequirements(b.catalog, modulePlans)
		if prefetchErr := b.runtimes.WarmIndexes(ctx, requirements); prefetchErr != nil {
			log.WithError(prefetchErr).Warn("failed to prefetch some pricing data")
		}
	}

	sem := make(chan struct{}, maxModuleConcurrency)
	var wg sync.WaitGroup
	for _, scanned := range executablePlans {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer func() { <-sem; wg.Done() }()
			moduleResults[scanned.Index] = *b.executor.Execute(ctx, scanned.Plan)
		}()
	}
	wg.Wait()

	assembler := results.NewEstimateAssembler(b.providerMetadata(), time.Now().UTC())
	for i := range moduleResults {
		assembler.AddModule(moduleResults[i])
	}

	return assembler.Build(), nil
}

// buildPrefetchRequirements analyses scanned module plans and returns the set of
// service/region pricing indexes that must be warm before estimation can run.
func buildPrefetchRequirements(catalog costruntime.ProviderCatalogRuntime, modulePlans []*ModulePlan) map[pricing.ServiceID][]string {
	regionSet := make(map[pricing.ServiceID]map[string]struct{})

	for _, modulePlan := range modulePlans {
		for _, resource := range modulePlan.Resources {
			providerID, ok := catalog.ResolveProvider(resource.ResourceType)
			if !ok {
				continue
			}

			h, ok := catalog.ResolveHandler(providerID, resource.ResourceType)
			if !ok || h.Category() != handler.CostCategoryStandard {
				continue
			}

			lb, ok := h.(handler.LookupBuilder)
			if !ok {
				continue
			}

			lookup, err := lb.BuildLookup(modulePlan.Region, resource.ActiveAttrs())
			if err != nil || lookup == nil {
				continue
			}

			if regionSet[lookup.ServiceID] == nil {
				regionSet[lookup.ServiceID] = make(map[string]struct{})
			}
			regionSet[lookup.ServiceID][modulePlan.Region] = struct{}{}
		}
	}

	services := make(map[pricing.ServiceID][]string, len(regionSet))
	for svc, regions := range regionSet {
		for region := range regions {
			services[svc] = append(services[svc], region)
		}
	}
	return services
}
