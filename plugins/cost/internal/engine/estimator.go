package engine

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	costruntime "github.com/edelwud/terraci/plugins/cost/internal/runtime"
)

// Estimator calculates cost estimates for terraform plans.
type Estimator struct {
	coord   *estimateCoordinator
	runtime *costruntime.EstimationRuntime
}

// NewEstimatorFromConfig creates an Estimator from config and a prepared blob cache.
// This is the canonical production constructor.
func NewEstimatorFromConfig(cfg *model.CostConfig, cache *blobcache.Cache) (*Estimator, error) {
	providers, err := configuredProviders(cfg)
	if err != nil {
		return nil, fmt.Errorf("resolve cost providers: %w", err)
	}
	runtime, err := costruntime.NewEstimationRuntimeFromProviders(providers, cache, nil)
	if err != nil {
		return nil, fmt.Errorf("create estimation runtime: %w", err)
	}
	return newEstimator(runtime)
}

// NewEstimatorWithDeps creates an Estimator with an explicit combined runtime dependency.
// Use for testing or advanced DI scenarios where exact provider wiring must be controlled.
func NewEstimatorWithDeps(runtime *costruntime.EstimationRuntime) (*Estimator, error) {
	return newEstimator(runtime)
}

// newEstimator is the internal constructor that wires all engine components.
func newEstimator(runtime *costruntime.EstimationRuntime) (*Estimator, error) {
	if runtime == nil {
		return nil, errors.New("cost estimator: estimation runtime is required")
	}
	resolver, err := costruntime.NewCostResolver(runtime, runtime)
	if err != nil {
		return nil, fmt.Errorf("create cost resolver: %w", err)
	}
	scanner := NewModuleScanner(NewTerraformPlanAdapter())
	executor := NewModuleExecutor(resolver)
	coord := newEstimateCoordinator(scanner, executor, runtime)

	return &Estimator{
		coord:   coord,
		runtime: runtime,
	}, nil
}

// configuredProviders resolves the subset of registered cloud providers enabled by config.
// Matches enabled config keys against cloud.Definition.ConfigKey — no hardcoded provider names.
func configuredProviders(cfg *model.CostConfig) ([]cloud.Provider, error) {
	enabled := map[string]struct{}{}
	if cfg != nil {
		for id, p := range cfg.Providers {
			if p.Enabled {
				enabled[id] = struct{}{}
			}
		}
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

// Cache returns a CacheInspector for diagnostic and maintenance access to the pricing cache.
func (e *Estimator) Cache() CacheInspector { return &cacheInspector{r: e.runtime} }

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
