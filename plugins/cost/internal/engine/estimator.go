package engine

import (
	"context"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
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
	catalog  *costruntime.ProviderCatalog
	runtimes *costruntime.ProviderRuntimeRegistry
	prefetch *costruntime.PricingPrefetcher
}

// NewEstimator creates a new cost estimator with the given blob store and pricing fetcher.
func NewEstimator(store plugin.BlobStore, cacheNamespace string, cacheTTL time.Duration, fetcher pricing.PriceFetcher) *Estimator {
	if cacheTTL == 0 {
		cacheTTL = pricing.DefaultCacheTTL
	}

	return NewEstimatorWithBlobCache(blobcache.New(store, cacheNamespace, cacheTTL), fetcher)
}

// NewEstimatorWithBlobCache creates a new cost estimator over a prepared blob cache.
func NewEstimatorWithBlobCache(cache *blobcache.Cache, fetcher pricing.PriceFetcher) *Estimator {
	providers := cloud.Providers()
	registry := newDefaultRegistry(providers)
	catalog := costruntime.NewProviderCatalogFromProviders(providers, registry)
	runtimeRegistry := costruntime.NewProviderRuntimeRegistryFromProvidersWithBlobCache(providers, cache, fetcher)
	return NewEstimatorWithCatalogAndRuntimeRegistry(catalog, runtimeRegistry)
}

// NewEstimatorWithRuntimeRegistry creates an Estimator with an explicit provider runtime registry.
func NewEstimatorWithRuntimeRegistry(runtimeRegistry *costruntime.ProviderRuntimeRegistry) *Estimator {
	providers := cloud.Providers()
	registry := newDefaultRegistry(providers)
	catalog := costruntime.NewProviderCatalogFromProviders(providers, registry)
	return NewEstimatorWithCatalogAndRuntimeRegistry(catalog, runtimeRegistry)
}

// NewEstimatorWithCatalogAndRuntimeRegistry creates an Estimator with explicit catalog and runtime dependencies.
func NewEstimatorWithCatalogAndRuntimeRegistry(catalog *costruntime.ProviderCatalog, runtimeRegistry *costruntime.ProviderRuntimeRegistry) *Estimator {
	resolutionRuntime := costruntime.NewResolutionRuntime(catalog, runtimeRegistry)
	resolver := costruntime.NewCostResolver(resolutionRuntime)
	return newEstimator(catalog, runtimeRegistry, resolver)
}

// NewEstimatorWithResolver creates an estimator with explicit catalog, runtime registry, and resolver.
func NewEstimatorWithResolver(catalog *costruntime.ProviderCatalog, runtimeRegistry *costruntime.ProviderRuntimeRegistry, resolver *costruntime.CostResolver) *Estimator {
	return newEstimator(catalog, runtimeRegistry, resolver)
}

func newEstimator(catalog *costruntime.ProviderCatalog, runtimeRegistry *costruntime.ProviderRuntimeRegistry, resolver *costruntime.CostResolver) *Estimator {
	scanner := NewModuleScanner(NewTerraformPlanAdapter())
	executor := NewModuleExecutor(resolver)
	planner := NewPrefetchPlanner(catalog)
	prefetcher := costruntime.NewPricingPrefetcher(runtimeRegistry)
	coord := newEstimateCoordinator(scanner, planner, executor, prefetcher, catalog.ProviderMetadata)

	return &Estimator{
		resolver: resolver,
		scanner:  scanner,
		executor: executor,
		planner:  planner,
		coord:    coord,
		catalog:  catalog,
		runtimes: runtimeRegistry,
		prefetch: prefetcher,
	}
}

// CacheDir returns the resolved pricing cache directory path.
func (e *Estimator) CacheDir() string { return e.runtimes.CacheDir() }

// SetPricingFetcher replaces the pricing fetcher (for testing or alternative providers).
func (e *Estimator) SetPricingFetcher(f pricing.PriceFetcher) { e.runtimes.SetPricingFetcher(f) }

// CacheOldestAge returns the age of the oldest cache entry, or 0 if empty.
func (e *Estimator) CacheOldestAge(ctx context.Context) time.Duration {
	return e.runtimes.CacheOldestAge(ctx)
}

// CacheTTL returns the cache TTL.
func (e *Estimator) CacheTTL() time.Duration { return e.runtimes.CacheTTL() }

// CleanExpiredCache removes expired cache entries.
func (e *Estimator) CleanExpiredCache(ctx context.Context) { e.runtimes.CleanExpiredCache(ctx) }

// CacheEntries returns info about all cached pricing files.
func (e *Estimator) CacheEntries(ctx context.Context) []pricing.CacheEntry {
	return e.runtimes.CacheEntries(ctx)
}

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
	return e.prefetch.PrefetchPricing(ctx, e.planner.Build(modulePlans))
}
