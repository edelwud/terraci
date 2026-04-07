package engine

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

// Estimator calculates cost estimates for terraform plans.
type Estimator struct {
	coord    *estimateCoordinator
	runtimes *costruntime.ProviderRuntimeRegistry
}

// NewEstimatorFromConfig creates an Estimator from config and a prepared blob cache.
// This is the canonical production constructor.
func NewEstimatorFromConfig(cfg *model.CostConfig, cache *blobcache.Cache) (*Estimator, error) {
	providers, err := configuredProviders(cfg)
	if err != nil {
		return nil, fmt.Errorf("resolve cost providers: %w", err)
	}
	registry := buildHandlerRegistry(providers)
	catalog := costruntime.NewProviderCatalogFromProviders(providers, registry)
	runtimeRegistry := costruntime.NewProviderRuntimeRegistryFromProviders(providers, cache, nil)
	return newEstimator(catalog, runtimeRegistry), nil
}

// NewEstimatorWithDeps creates an Estimator with explicit catalog and runtime dependencies.
// Use for testing or advanced DI scenarios where exact provider wiring must be controlled.
func NewEstimatorWithDeps(catalog *costruntime.ProviderCatalog, runtimeRegistry *costruntime.ProviderRuntimeRegistry) *Estimator {
	return newEstimator(catalog, runtimeRegistry)
}

// newEstimator is the internal constructor that wires all engine components.
func newEstimator(catalog *costruntime.ProviderCatalog, runtimeRegistry *costruntime.ProviderRuntimeRegistry) *Estimator {
	resolver := costruntime.NewCostResolver(costruntime.CombineRuntime(catalog, runtimeRegistry))
	scanner := NewModuleScanner(NewTerraformPlanAdapter())
	executor := NewModuleExecutor(resolver)
	coord := newEstimateCoordinator(scanner, executor, catalog, catalog.ProviderMetadata, runtimeRegistry)

	return &Estimator{
		coord:    coord,
		runtimes: runtimeRegistry,
	}
}

// enabledProviderIDs returns IDs of all cloud providers enabled in the config.
// Iterates the provider config map — no knowledge of specific provider IDs required.
func enabledProviderIDs(cfg *model.CostConfig) []string {
	if cfg == nil {
		return nil
	}
	var ids []string
	for id, p := range cfg.Providers {
		if p.Enabled {
			ids = append(ids, id)
		}
	}
	return ids
}

// configuredProviders resolves the subset of registered cloud providers enabled by config.
// Matches enabled config keys against cloud.Definition.ConfigKey — no hardcoded provider names.
func configuredProviders(cfg *model.CostConfig) ([]cloud.Provider, error) {
	enabled := map[string]struct{}{}
	for _, id := range enabledProviderIDs(cfg) {
		enabled[id] = struct{}{}
	}

	all := cloud.Providers()
	selected := make([]cloud.Provider, 0, len(enabled))
	for _, cp := range all {
		if _, ok := enabled[cp.Definition().ConfigKey]; ok {
			selected = append(selected, cp)
		}
	}

	for id := range enabled {
		found := false
		for _, cp := range all {
			if cp.Definition().ConfigKey == id {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("cost provider %q is enabled but not registered", id)
		}
	}

	return selected, nil
}

// buildHandlerRegistry populates a handler registry from provider definitions.
func buildHandlerRegistry(providers []cloud.Provider) *handler.Registry {
	r := handler.NewRegistry()
	for _, cp := range providers {
		cloud.RegisterDefinitionHandlers(r, cp.Definition())
	}
	return r
}

// Cache returns a CacheInspector for diagnostic and maintenance access to the pricing cache.
func (e *Estimator) Cache() CacheInspector { return &cacheInspector{r: e.runtimes} }

// SetFetcherForProvider replaces the pricing fetcher for a specific provider.
// Use only in tests to inject a stub fetcher before running estimates.
func (e *Estimator) SetFetcherForProvider(providerID string, f pricing.PriceFetcher) {
	e.runtimes.SetFetcherForProvider(providerID, f)
}

// EstimateModule calculates cost for a single module from plan.json.
func (e *Estimator) EstimateModule(ctx context.Context, modulePath, region string) (*model.ModuleCost, error) {
	modulePlan, err := e.coord.scanner.Scan(modulePath, region)
	if err != nil {
		return nil, err
	}
	return e.coord.executor.Execute(ctx, modulePlan), nil
}

// EstimateModules calculates costs for multiple modules concurrently.
func (e *Estimator) EstimateModules(ctx context.Context, modulePaths []string, regions map[string]string) (*model.EstimateResult, error) {
	return e.coord.Estimate(ctx, modulePaths, regions)
}
