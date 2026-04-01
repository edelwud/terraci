package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ProviderRuntimeRegistry is the pricing/cache runtime surface used by estimation components.
type ProviderRuntimeRegistry struct {
	runtimes map[string]*ProviderRuntime
	cache    *blobcache.Cache
	inspect  *pricing.CacheInspector
}

// ProviderRuntime groups the provider registry entry with its pricing cache.
type ProviderRuntime struct {
	Definition cloud.Definition
	Cache      *pricing.Cache
}

// NewProviderRuntimeRegistry creates a provider runtime registry from explicit provider runtimes.
func NewProviderRuntimeRegistry(runtimes map[string]*ProviderRuntime) *ProviderRuntimeRegistry {
	return &ProviderRuntimeRegistry{runtimes: runtimes}
}

// NewProviderRuntimeRegistryFromProviders creates a runtime registry directly from provider definitions.
func NewProviderRuntimeRegistryFromProviders(
	providers []cloud.Provider,
	store plugin.BlobStore,
	cacheNamespace string,
	cacheTTL time.Duration,
	fetcher pricing.PriceFetcher,
) *ProviderRuntimeRegistry {
	if cacheTTL == 0 {
		cacheTTL = pricing.DefaultCacheTTL
	}

	return NewProviderRuntimeRegistryFromProvidersWithBlobCache(
		providers,
		blobcache.New(store, cacheNamespace, cacheTTL),
		fetcher,
	)
}

// NewProviderRuntimeRegistryFromProvidersWithBlobCache creates a runtime registry from provider
// definitions over a prepared blob cache.
func NewProviderRuntimeRegistryFromProvidersWithBlobCache(
	providers []cloud.Provider,
	cache *blobcache.Cache,
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
			Cache:      pricing.NewCacheFromBlobCache(cache, runtimeFetcher),
		}
	}
	return &ProviderRuntimeRegistry{
		runtimes: runtimes,
		cache:    cache,
		inspect:  pricing.NewCacheInspector(cache),
	}
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

// CacheDir returns the resolved pricing cache directory path.
func (r *ProviderRuntimeRegistry) CacheDir() string {
	if r.inspect == nil {
		return ""
	}
	return r.inspect.Dir()
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
func (r *ProviderRuntimeRegistry) CacheOldestAge(ctx context.Context) time.Duration {
	if r.inspect == nil {
		return 0
	}
	return r.inspect.OldestAge(ctx)
}

// CacheTTL returns the cache TTL.
func (r *ProviderRuntimeRegistry) CacheTTL() time.Duration {
	if r.inspect == nil {
		return 0
	}
	return r.inspect.TTL()
}

// CleanExpiredCache removes expired cache entries.
func (r *ProviderRuntimeRegistry) CleanExpiredCache(ctx context.Context) {
	if r.cache == nil {
		return
	}
	if err := r.cache.CleanExpired(ctx); err != nil {
		log.WithError(err).Debug("failed to clean expired cache")
	}
}

// CacheEntries returns info about all cached pricing files.
func (r *ProviderRuntimeRegistry) CacheEntries(ctx context.Context) []pricing.CacheEntry {
	if r.inspect == nil {
		return nil
	}
	return r.inspect.Entries(ctx)
}
