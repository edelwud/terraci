package costengine

import (
	"time"

	// Blank import triggers AWS provider self-registration via init().
	// Add similar imports for GCP/Azure when available.
	_ "github.com/edelwud/terraci/plugins/cost/internal/cloud/aws"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const defaultCacheTTL = 24 * time.Hour

// newDefaultRegistry creates a handler registry with all registered cloud providers' handlers.
func newDefaultRegistry() *handler.Registry {
	r := handler.NewRegistry()
	for _, cp := range cloud.Providers() {
		cp.InitRegionMapping()
		cp.RegisterHandlers(r)
	}
	return r
}

// newDefaultFetcher returns a composite fetcher from registered cloud providers.
func newDefaultFetcher() pricing.PriceFetcher {
	providers := cloud.Providers()
	if len(providers) == 0 {
		return nil
	}
	if len(providers) == 1 {
		return providers[0].NewFetcher()
	}
	fetchers := make(map[string]pricing.PriceFetcher, len(providers))
	for _, cp := range providers {
		fetchers[cp.Name()] = cp.NewFetcher()
	}
	return &cloud.RoutingFetcher{Fetchers: fetchers}
}

// NewEstimatorFromConfig creates an Estimator using CostConfig settings.
// Uses all registered cloud providers (registered via init() + cloud.Register).
func NewEstimatorFromConfig(cfg *CostConfig) *Estimator {
	cacheDir, cacheTTL := parseCacheConfig(cfg)

	registry := newDefaultRegistry()
	fetcher := newDefaultFetcher()
	cache := pricing.NewCache(cacheDir, cacheTTL, fetcher)
	resolver := NewCostResolver(registry, cache)
	return NewEstimatorWithResolver(cache, resolver)
}

// NewEstimatorFromConfigWithProvider creates an Estimator for a specific cloud provider.
// Use this to override auto-discovery for testing or single-provider usage.
func NewEstimatorFromConfigWithProvider(cfg *CostConfig, cp cloud.Provider) *Estimator {
	cacheDir, cacheTTL := parseCacheConfig(cfg)

	cp.InitRegionMapping()
	registry := handler.NewRegistry()
	cp.RegisterHandlers(registry)

	fetcher := cp.NewFetcher()
	cache := pricing.NewCache(cacheDir, cacheTTL, fetcher)
	resolver := NewCostResolver(registry, cache)
	return NewEstimatorWithResolver(cache, resolver)
}

// parseCacheConfig extracts cache directory and TTL from config.
func parseCacheConfig(cfg *CostConfig) (string, time.Duration) {
	cacheDir := ""
	cacheTTL := defaultCacheTTL

	if cfg != nil {
		if cfg.CacheDir != "" {
			cacheDir = cfg.CacheDir
		}
		if cfg.CacheTTL != "" {
			if d, err := time.ParseDuration(cfg.CacheTTL); err == nil {
				cacheTTL = d
			}
		}
	}

	return cacheDir, cacheTTL
}
