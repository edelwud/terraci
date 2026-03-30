package engine

import (
	"fmt"
	"time"

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

func newProviderRuntimeRegistry(cfg *model.CostConfig, providers []cloud.Provider) *costruntime.ProviderRuntimeRegistry {
	cacheDir, cacheTTL := parseCacheConfig(cfg)
	return costruntime.NewProviderRuntimeRegistryFromProviders(providers, cacheDir, cacheTTL, nil)
}

// NewEstimatorFromConfig creates an Estimator using CostConfig settings.
func NewEstimatorFromConfig(cfg *model.CostConfig) (*Estimator, error) {
	providers, err := configuredProviders(cfg)
	if err != nil {
		return nil, err
	}

	registry := newDefaultRegistry(providers)
	runtimeRegistry := newProviderRuntimeRegistry(cfg, providers)
	return NewEstimatorWithRuntimeRegistry(registry, runtimeRegistry), nil
}

// NewEstimatorFromConfigWithProvider creates an Estimator for a specific cloud provider.
func NewEstimatorFromConfigWithProvider(cfg *model.CostConfig, cp cloud.Provider) *Estimator {
	registry := handler.NewRegistry()
	cloud.RegisterDefinitionHandlers(registry, cp.Definition())
	runtimeRegistry := newProviderRuntimeRegistry(cfg, []cloud.Provider{cp})
	return NewEstimatorWithRuntimeRegistry(registry, runtimeRegistry)
}

func parseCacheConfig(cfg *model.CostConfig) (string, time.Duration) {
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
