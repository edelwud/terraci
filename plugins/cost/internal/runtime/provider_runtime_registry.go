package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ProviderRouter resolves the owning cloud provider for a Terraform resource type.
type ProviderRouter interface {
	ResolveProvider(resourceType handler.ResourceType) (string, bool)
}

// ResourceProviderRouter is the default resource-type based provider router.
type ResourceProviderRouter struct {
	providers map[handler.ResourceType]string
}

// NewResourceProviderRouter creates an empty provider router.
func NewResourceProviderRouter() *ResourceProviderRouter {
	return &ResourceProviderRouter{providers: make(map[handler.ResourceType]string)}
}

// Register records the owning provider for a resource type.
func (r *ResourceProviderRouter) Register(providerID string, resourceType handler.ResourceType) {
	r.providers[resourceType] = providerID
}

// ResolveProvider returns the provider id for a resource type.
func (r *ResourceProviderRouter) ResolveProvider(resourceType handler.ResourceType) (string, bool) {
	providerID, ok := r.providers[resourceType]
	return providerID, ok
}

func newDefaultProviderRouter(providers []cloud.Provider) *ResourceProviderRouter {
	router := NewResourceProviderRouter()
	for _, cp := range providers {
		def := cp.Definition()
		for _, resource := range def.Resources {
			router.Register(def.Manifest.ID, resource.Type)
		}
	}
	return router
}

// ProviderRuntimeRegistry is the provider-owned runtime surface used by estimation components.
type ProviderRuntimeRegistry struct {
	router   *ResourceProviderRouter
	runtimes map[string]*ProviderRuntime
}

// ProviderRuntime groups the provider registry entry with its pricing cache.
type ProviderRuntime struct {
	Definition cloud.Definition
	Cache      *pricing.Cache
}

// NewProviderRuntimeRegistry creates a provider runtime registry from provider definitions and runtimes.
func NewProviderRuntimeRegistry(providers []cloud.Provider, runtimes map[string]*ProviderRuntime) *ProviderRuntimeRegistry {
	return &ProviderRuntimeRegistry{
		router:   newDefaultProviderRouter(providers),
		runtimes: runtimes,
	}
}

// NewProviderRuntimeRegistryFromProviders creates a runtime registry directly from provider definitions.
func NewProviderRuntimeRegistryFromProviders(
	providers []cloud.Provider,
	cacheDir string,
	cacheTTL time.Duration,
	fetcher pricing.PriceFetcher,
) *ProviderRuntimeRegistry {
	runtimes := make(map[string]*ProviderRuntime, len(providers))
	for _, cp := range providers {
		def := cp.Definition()
		runtimeFetcher := def.FetcherFactory()
		if len(providers) == 1 && fetcher != nil {
			runtimeFetcher = fetcher
		}
		runtimes[def.Manifest.ID] = &ProviderRuntime{
			Definition: def,
			Cache:      pricing.NewCache(cacheDir, cacheTTL, runtimeFetcher),
		}
	}
	return NewProviderRuntimeRegistry(providers, runtimes)
}

// ResolveProvider returns the owning provider for a resource type.
func (r *ProviderRuntimeRegistry) ResolveProvider(resourceType handler.ResourceType) (string, bool) {
	return r.router.ResolveProvider(resourceType)
}

// SetRouter replaces the provider router.
func (r *ProviderRuntimeRegistry) SetRouter(router *ResourceProviderRouter) {
	r.router = router
}

func (r *ProviderRuntimeRegistry) getRuntime(providerID string) (*ProviderRuntime, bool) {
	runtime, ok := r.runtimes[providerID]
	return runtime, ok
}

// GetIndex resolves pricing through the runtime selected by service id.
func (r *ProviderRuntimeRegistry) GetIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
	runtime, ok := r.getRuntime(service.Provider)
	if !ok {
		return nil, fmt.Errorf("no pricing runtime for provider %q", service.Provider)
	}
	return runtime.Cache.GetIndex(ctx, service, region)
}

// SourceName returns the configured price source for a provider.
func (r *ProviderRuntimeRegistry) SourceName(providerID string) string {
	runtime, ok := r.getRuntime(providerID)
	if !ok {
		return ""
	}
	return runtime.Definition.Manifest.PriceSource
}

// ProviderMetadata returns provider-specific estimation metadata keyed by provider id.
func (r *ProviderRuntimeRegistry) ProviderMetadata() map[string]model.ProviderMetadata {
	if len(r.runtimes) == 0 {
		return nil
	}

	meta := make(map[string]model.ProviderMetadata, len(r.runtimes))
	for providerID, runtime := range r.runtimes {
		if runtime.Definition.Manifest.ID == "" {
			continue
		}
		meta[providerID] = model.ProviderMetadata{
			DisplayName: runtime.Definition.Manifest.DisplayName,
			PriceSource: runtime.Definition.Manifest.PriceSource,
		}
	}
	return meta
}

// CacheDir returns the resolved pricing cache directory path.
func (r *ProviderRuntimeRegistry) CacheDir() string {
	for _, runtime := range r.runtimes {
		return runtime.Cache.Dir()
	}
	return ""
}

// SetPricingFetcher replaces the pricing fetcher.
func (r *ProviderRuntimeRegistry) SetPricingFetcher(f pricing.PriceFetcher) {
	if len(r.runtimes) == 1 {
		for _, runtime := range r.runtimes {
			runtime.Cache.SetFetcher(f)
			return
		}
	}
	if runtime, ok := r.runtimes[awskit.ProviderID]; ok {
		runtime.Cache.SetFetcher(f)
	}
}

// CacheOldestAge returns the age of the oldest cache entry, or 0 if empty.
func (r *ProviderRuntimeRegistry) CacheOldestAge() time.Duration {
	oldest := time.Duration(0)
	for _, runtime := range r.runtimes {
		age := runtime.Cache.OldestAge()
		if oldest == 0 || (age != 0 && age > oldest) {
			oldest = age
		}
	}
	return oldest
}

// CacheTTL returns the cache TTL.
func (r *ProviderRuntimeRegistry) CacheTTL() time.Duration {
	for _, runtime := range r.runtimes {
		return runtime.Cache.TTL()
	}
	return 0
}

// CleanExpiredCache removes expired cache entries.
func (r *ProviderRuntimeRegistry) CleanExpiredCache() {
	for providerID, runtime := range r.runtimes {
		if err := runtime.Cache.CleanExpired(); err != nil {
			log.WithError(err).WithField("provider", providerID).Debug("failed to clean expired cache")
		}
	}
}

// CacheEntries returns info about all cached pricing files.
func (r *ProviderRuntimeRegistry) CacheEntries() []pricing.CacheEntry {
	var entries []pricing.CacheEntry
	for _, runtime := range r.runtimes {
		entries = append(entries, runtime.Cache.Entries()...)
	}
	return entries
}

// ServicePlan exposes the minimal prefetch-plan shape consumed by the runtime registry.
type ServicePlan interface {
	Services() map[pricing.ServiceID][]string
}

// PrefetchPricing downloads any missing pricing data required by the plan.
func (r *ProviderRuntimeRegistry) PrefetchPricing(ctx context.Context, prefetchPlan ServicePlan) error {
	services := prefetchPlan.Services()
	if len(services) == 0 {
		log.Warn("no supported resources found in plans - nothing to price")
		return nil
	}

	var totalMissing int
	for providerID, runtime := range r.runtimes {
		providerServices := filterServicesForProvider(services, providerID)
		if len(providerServices) == 0 {
			continue
		}
		missing := runtime.Cache.Validate(providerServices)
		if len(missing) == 0 {
			continue
		}
		for i, m := range missing {
			totalMissing++
			log.WithField("provider", providerID).
				WithField("service", m.Service.String()).
				WithField("region", m.Region).
				WithField("progress", fmt.Sprintf("%d/%d", i+1, len(missing))).
				Info("downloading pricing data")
			if _, err := runtime.Cache.GetIndex(ctx, m.Service, m.Region); err != nil {
				return fmt.Errorf("fetch %s/%s pricing: %w", m.Service.String(), m.Region, err)
			}
		}
	}

	if totalMissing == 0 {
		log.Info("all pricing data is cached and valid")
	}

	return nil
}

func filterServicesForProvider(services map[pricing.ServiceID][]string, providerID string) map[pricing.ServiceID][]string {
	filtered := make(map[pricing.ServiceID][]string)
	for serviceID, regions := range services {
		if serviceID.Provider == providerID {
			filtered[serviceID] = regions
		}
	}
	return filtered
}
