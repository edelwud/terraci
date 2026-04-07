package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
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
		// fetcher override is only supported for single-provider setups (typically tests).
		// For multi-provider setups, pass nil and use SetFetcherForProvider per provider.
		if fetcher != nil {
			if len(providers) != 1 {
				panic("runtime: fetcher override requires exactly one provider; use SetFetcherForProvider for multi-provider setups")
			}
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

// SetFetcherForProvider replaces the pricing fetcher for a specific provider.
// Intended for tests only: inject a stub fetcher without building a full runtime.
// Silent no-op if providerID is not registered (multi-provider setups must call
// this for each registered provider separately).
func (r *ProviderRuntimeRegistry) SetFetcherForProvider(providerID string, f pricing.PriceFetcher) {
	if rt, ok := r.runtimes[providerID]; ok {
		rt.Cache.SetFetcher(f)
	}
}

// WarmIndexes downloads any missing pricing data for the given service/region requirements.
// Warming is best-effort: all service/region pairs are attempted even if some fail.
// Returns a joined error if one or more downloads fail.
func (r *ProviderRuntimeRegistry) WarmIndexes(ctx context.Context, services map[pricing.ServiceID][]string) error {
	if len(services) == 0 {
		return nil
	}
	var errs []error
	for serviceID, regions := range services {
		rt, ok := r.runtimes[serviceID.Provider]
		if !ok {
			continue
		}
		for _, region := range regions {
			if _, err := rt.Cache.GetIndex(ctx, serviceID, region); err != nil {
				errs = append(errs, fmt.Errorf("%s/%s: %w", serviceID, region, err))
			}
		}
	}
	return errors.Join(errs...)
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
	if r.inspect == nil {
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
