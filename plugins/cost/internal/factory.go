package costengine

import (
	"fmt"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
)

const defaultCacheTTL = 24 * time.Hour

// newDefaultRegistry creates a handler registry with all enabled cloud providers' handlers.
func newDefaultRegistry(providers []cloud.Provider) *handler.Registry {
	r := handler.NewRegistry()
	for _, cp := range providers {
		cloud.RegisterDefinitionHandlers(r, cp.Definition())
	}
	return r
}

func configuredProviders(cfg *CostConfig) ([]cloud.Provider, error) {
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

func newProviderRuntimeRegistry(cfg *CostConfig, providers []cloud.Provider) *ProviderRuntimeRegistry {
	cacheDir, cacheTTL := parseCacheConfig(cfg)
	return NewProviderRuntimeRegistryFromProviders(providers, cacheDir, cacheTTL, nil)
}

// NewEstimatorFromConfig creates an Estimator using CostConfig settings.
// Uses the configured cloud providers registered via init() + cloud.Register.
func NewEstimatorFromConfig(cfg *CostConfig) (*Estimator, error) {
	providers, err := configuredProviders(cfg)
	if err != nil {
		return nil, err
	}

	registry := newDefaultRegistry(providers)
	runtimeRegistry := newProviderRuntimeRegistry(cfg, providers)
	return NewEstimatorWithRuntimeRegistry(registry, runtimeRegistry), nil
}

// NewEstimatorFromConfigWithProvider creates an Estimator for a specific cloud provider.
// Use this to override auto-discovery for testing or single-provider usage.
func NewEstimatorFromConfigWithProvider(cfg *CostConfig, cp cloud.Provider) *Estimator {
	registry := handler.NewRegistry()
	cloud.RegisterDefinitionHandlers(registry, cp.Definition())
	runtimeRegistry := newProviderRuntimeRegistry(cfg, []cloud.Provider{cp})
	return NewEstimatorWithRuntimeRegistry(registry, runtimeRegistry)
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
