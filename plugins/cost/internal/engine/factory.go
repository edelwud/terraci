package engine

import (
	"fmt"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

const defaultCacheTTL = 24 * time.Hour

func newDefaultRegistry(providers []cloud.Provider) *handler.Registry {
	r := handler.NewRegistry()
	for _, cp := range providers {
		cloud.RegisterDefinitionHandlers(r, cp.Definition())
	}
	return r
}

func configuredProviders(cfg *model.CostConfig) ([]cloud.Provider, error) {
	enabled := map[string]bool{}
	for _, id := range cfg.EnabledProviderIDs() {
		enabled[id] = true
	}

	providers := cloud.Providers()
	selected := make([]cloud.Provider, 0, len(enabled))
	for _, cp := range providers {
		if enabled[cp.Definition().Manifest.ID] {
			selected = append(selected, cp)
		}
	}

	for id := range enabled {
		if _, ok := cloud.Get(id); !ok {
			return nil, fmt.Errorf("cost provider %q is enabled but not registered", id)
		}
	}

	return selected, nil
}

func newProviderCatalog(providers []cloud.Provider, registry *handler.Registry) *costruntime.ProviderCatalog {
	return costruntime.NewProviderCatalogFromProviders(providers, registry)
}

// NewEstimatorFromConfig creates an Estimator using CostConfig settings and the resolved blob store.
func NewEstimatorFromConfig(cfg *model.CostConfig, store plugin.BlobStore) (*Estimator, error) {
	return NewEstimatorFromConfigWithBlobCache(cfg, blobcache.New(store, cfg.BlobCacheNamespace(), CacheTTLFromConfig(cfg)))
}

// NewEstimatorFromConfigWithBlobCache creates an Estimator using CostConfig settings and a prepared blob cache.
func NewEstimatorFromConfigWithBlobCache(cfg *model.CostConfig, cache *blobcache.Cache) (*Estimator, error) {
	providers, err := configuredProviders(cfg)
	if err != nil {
		return nil, err
	}

	registry := newDefaultRegistry(providers)
	catalog := newProviderCatalog(providers, registry)
	runtimeRegistry := costruntime.NewProviderRuntimeRegistryFromProvidersWithBlobCache(providers, cache, nil)
	return NewEstimatorWithCatalogAndRuntimeRegistry(catalog, runtimeRegistry), nil
}

// NewEstimatorFromConfigWithProvider creates an Estimator for a specific cloud provider.
func NewEstimatorFromConfigWithProvider(cfg *model.CostConfig, cp cloud.Provider, store plugin.BlobStore) *Estimator {
	return NewEstimatorFromConfigWithProviderAndBlobCache(cp, blobcache.New(store, cfg.BlobCacheNamespace(), CacheTTLFromConfig(cfg)))
}

// NewEstimatorFromConfigWithProviderAndBlobCache creates an Estimator for a specific cloud provider
// over a prepared blob cache.
func NewEstimatorFromConfigWithProviderAndBlobCache(cp cloud.Provider, cache *blobcache.Cache) *Estimator {
	registry := handler.NewRegistry()
	cloud.RegisterDefinitionHandlers(registry, cp.Definition())
	catalog := newProviderCatalog([]cloud.Provider{cp}, registry)
	runtimeRegistry := costruntime.NewProviderRuntimeRegistryFromProvidersWithBlobCache([]cloud.Provider{cp}, cache, nil)
	return NewEstimatorWithCatalogAndRuntimeRegistry(catalog, runtimeRegistry)
}

// CacheTTLFromConfig resolves the configured pricing cache TTL or the built-in default.
func CacheTTLFromConfig(cfg *model.CostConfig) time.Duration {
	cacheTTL := defaultCacheTTL

	if cfg != nil {
		if ttl := cfg.BlobCacheTTL(); ttl != "" {
			if d, err := time.ParseDuration(ttl); err == nil {
				cacheTTL = d
			}
		}
	}

	return cacheTTL
}
