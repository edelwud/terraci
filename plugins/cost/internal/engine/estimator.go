package engine

import (
	"context"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

// Estimator calculates cost estimates for terraform plans.
type Estimator struct {
	resolver *costruntime.CostResolver
	scanner  *ModuleScanner
	executor *ModuleExecutor
	planner  *PrefetchPlanner
	coord    *estimateCoordinator
	runtimes *costruntime.ProviderRuntimeRegistry
}

// NewEstimator creates a new cost estimator with the given pricing fetcher.
func NewEstimator(cacheDir string, cacheTTL time.Duration, fetcher pricing.PriceFetcher) *Estimator {
	providers := cloud.Providers()
	registry := newDefaultRegistry(providers)
	runtimeRegistry := costruntime.NewProviderRuntimeRegistryFromProviders(providers, cacheDir, cacheTTL, fetcher)
	return NewEstimatorWithRuntimeRegistry(registry, runtimeRegistry)
}

// NewEstimatorWithRuntimeRegistry creates an Estimator with an explicit provider runtime registry.
func NewEstimatorWithRuntimeRegistry(registry *handler.Registry, runtimeRegistry *costruntime.ProviderRuntimeRegistry) *Estimator {
	resolver := costruntime.NewCostResolver(runtimeRegistry, registry, runtimeRegistry)
	return newEstimator(runtimeRegistry, resolver)
}

// NewEstimatorWithResolver creates an estimator with an explicit runtime registry and resolver.
func NewEstimatorWithResolver(runtimeRegistry *costruntime.ProviderRuntimeRegistry, resolver *costruntime.CostResolver) *Estimator {
	return newEstimator(runtimeRegistry, resolver)
}

func newEstimator(runtimeRegistry *costruntime.ProviderRuntimeRegistry, resolver *costruntime.CostResolver) *Estimator {
	scanner := NewModuleScanner(NewTerraformPlanAdapter())
	executor := NewModuleExecutor(resolver)
	planner := NewPrefetchPlanner(runtimeRegistry, resolver.Registry())
	coord := newEstimateCoordinator(scanner, planner, executor, runtimeRegistry, runtimeRegistry.ProviderMetadata)

	return &Estimator{
		resolver: resolver,
		scanner:  scanner,
		executor: executor,
		planner:  planner,
		coord:    coord,
		runtimes: runtimeRegistry,
	}
}

// CacheDir returns the resolved pricing cache directory path.
func (e *Estimator) CacheDir() string { return e.runtimes.CacheDir() }

// SetPricingFetcher replaces the pricing fetcher (for testing or alternative providers).
func (e *Estimator) SetPricingFetcher(f pricing.PriceFetcher) { e.runtimes.SetPricingFetcher(f) }

// CacheOldestAge returns the age of the oldest cache entry, or 0 if empty.
func (e *Estimator) CacheOldestAge() time.Duration { return e.runtimes.CacheOldestAge() }

// CacheTTL returns the cache TTL.
func (e *Estimator) CacheTTL() time.Duration { return e.runtimes.CacheTTL() }

// CleanExpiredCache removes expired cache entries.
func (e *Estimator) CleanExpiredCache() { e.runtimes.CleanExpiredCache() }

// CacheEntries returns info about all cached pricing files.
func (e *Estimator) CacheEntries() []pricing.CacheEntry { return e.runtimes.CacheEntries() }

// Resolver returns the underlying CostResolver for middleware registration.
func (e *Estimator) Resolver() *costruntime.CostResolver { return e.resolver }

// GetIndex resolves pricing through the provider runtime selected by service id.
func (e *Estimator) GetIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
	return e.runtimes.GetIndex(ctx, service, region)
}

// SourceName returns the configured price source for a provider.
func (e *Estimator) SourceName(providerID string) string { return e.runtimes.SourceName(providerID) }

// EstimateModule calculates cost for a single module from plan.json.
func (e *Estimator) EstimateModule(ctx context.Context, modulePath, region string) (*model.ModuleCost, error) {
	modulePlan, err := e.scanner.Scan(modulePath, region)
	if err != nil {
		return nil, err
	}
	return e.executor.Execute(ctx, modulePlan), nil
}

// EstimateModules calculates costs for multiple modules concurrently.
func (e *Estimator) EstimateModules(ctx context.Context, modulePaths []string, regions map[string]string) (*model.EstimateResult, error) {
	return e.coord.Estimate(ctx, modulePaths, regions)
}

// ValidateAndPrefetch checks which pricing data is needed and downloads missing data.
func (e *Estimator) ValidateAndPrefetch(ctx context.Context, modulePaths []string, regions map[string]string) error {
	modulePlans, err := e.scanner.ScanMany(modulePaths, regions)
	if err != nil {
		return err
	}
	return e.runtimes.PrefetchPricing(ctx, e.planner.Build(modulePlans))
}
