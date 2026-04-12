package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

// EstimationRuntime is the single engine-facing runtime surface for provider routing,
// handler lookup, pricing access, metadata, and cache diagnostics.
type EstimationRuntime struct {
	catalog *ProviderCatalog
	pricing *ProviderRuntimeRegistry
}

// NewEstimationRuntime creates a combined runtime from explicit catalog and pricing runtimes.
func NewEstimationRuntime(catalog *ProviderCatalog, pricingRuntime *ProviderRuntimeRegistry) (*EstimationRuntime, error) {
	if catalog == nil {
		return nil, errors.New("estimation runtime: provider catalog is required")
	}
	if pricingRuntime == nil {
		return nil, errors.New("estimation runtime: pricing runtime registry is required")
	}
	return &EstimationRuntime{
		catalog: catalog,
		pricing: pricingRuntime,
	}, nil
}

// NewEstimationRuntimeFromProviders creates the default runtime from provider definitions,
// handler registration, and pricing cache wiring.
func NewEstimationRuntimeFromProviders(
	providers []cloud.Provider,
	cache *blobcache.Cache,
	fetchers map[string]pricing.PriceFetcher,
) (*EstimationRuntime, error) {
	catalog := NewProviderCatalogFromProviders(providers)
	pricingRuntime, err := NewProviderRuntimeRegistryFromProviders(providers, cache, fetchers)
	if err != nil {
		return nil, fmt.Errorf("create pricing runtime registry: %w", err)
	}
	return NewEstimationRuntime(catalog, pricingRuntime)
}

func (r *EstimationRuntime) ResolveProvider(resourceType resourcedef.ResourceType) (string, bool) {
	return r.catalog.ResolveProvider(resourceType)
}

func (r *EstimationRuntime) ResolveDefinition(providerID string, resourceType resourcedef.ResourceType) (resourcedef.Definition, bool) {
	return r.catalog.ResolveDefinition(providerID, resourceType)
}

func (r *EstimationRuntime) ProviderMetadata() map[string]model.ProviderMetadata {
	return r.catalog.ProviderMetadata()
}

func (r *EstimationRuntime) GetIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
	return r.pricing.GetIndex(ctx, service, region)
}

func (r *EstimationRuntime) SourceName(providerID string) string {
	return r.pricing.SourceName(providerID)
}

func (r *EstimationRuntime) WarmIndexes(ctx context.Context, services map[pricing.ServiceID][]string) error {
	return r.pricing.WarmIndexes(ctx, services)
}

func (r *EstimationRuntime) CacheDir() string {
	return r.pricing.CacheDir()
}

func (r *EstimationRuntime) CacheTTL() time.Duration {
	return r.pricing.CacheTTL()
}

func (r *EstimationRuntime) CacheOldestAge(ctx context.Context) time.Duration {
	return r.pricing.CacheOldestAge(ctx)
}

func (r *EstimationRuntime) CacheEntries(ctx context.Context) []pricing.CacheEntry {
	return r.pricing.CacheEntries(ctx)
}

func (r *EstimationRuntime) CleanExpiredCache(ctx context.Context) {
	r.pricing.CleanExpiredCache(ctx)
}
