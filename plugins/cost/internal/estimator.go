package costengine

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// DefaultRegion is used when no region is specified.
const DefaultRegion = "us-east-1"

// Estimator calculates cost estimates for terraform plans.
type Estimator struct {
	resolver *CostResolver
	cache    *pricing.Cache
}

// NewEstimator creates a new cost estimator with the given pricing fetcher.
func NewEstimator(cacheDir string, cacheTTL time.Duration, fetcher pricing.PriceFetcher) *Estimator {
	registry := newDefaultRegistry()
	cache := pricing.NewCache(cacheDir, cacheTTL, fetcher)
	resolver := NewCostResolver(registry, cache)
	return NewEstimatorWithResolver(cache, resolver)
}

// NewEstimatorWithResolver creates an Estimator with an explicit resolver and cache.
// Use this for custom registries, middleware, or testing.
func NewEstimatorWithResolver(cache *pricing.Cache, resolver *CostResolver) *Estimator {
	return &Estimator{
		resolver: resolver,
		cache:    cache,
	}
}

// CacheDir returns the resolved pricing cache directory path.
func (e *Estimator) CacheDir() string { return e.cache.Dir() }

// SetPricingFetcher replaces the pricing fetcher (for testing or alternative providers).
func (e *Estimator) SetPricingFetcher(f pricing.PriceFetcher) { e.cache.SetFetcher(f) }

// CacheOldestAge returns the age of the oldest cache entry, or 0 if empty.
func (e *Estimator) CacheOldestAge() time.Duration { return e.cache.OldestAge() }

// CacheTTL returns the cache TTL.
func (e *Estimator) CacheTTL() time.Duration { return e.cache.TTL() }

// CleanExpiredCache removes expired cache entries.
func (e *Estimator) CleanExpiredCache() {
	if err := e.cache.CleanExpired(); err != nil {
		log.WithError(err).Debug("failed to clean expired cache")
	}
}

// CacheEntries returns info about all cached pricing files.
func (e *Estimator) CacheEntries() []pricing.CacheEntry { return e.cache.Entries() }

// Resolver returns the underlying CostResolver for middleware registration.
func (e *Estimator) Resolver() *CostResolver { return e.resolver }

// EstimateModule calculates cost for a single module from plan.json.
func (e *Estimator) EstimateModule(ctx context.Context, modulePath, region string) (*ModuleCost, error) {
	planJSONPath := filepath.Join(modulePath, "plan.json")

	parsedPlan, err := plan.ParseJSON(planJSONPath)
	if err != nil {
		return nil, fmt.Errorf("parse plan.json: %w", err)
	}

	result := &ModuleCost{
		ModuleID:   strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath: modulePath,
		Region:     region,
		Resources:  make([]ResourceCost, 0),
	}

	// Pre-fetch pricing data for all required services
	requiredServices := e.resolver.collectRequiredServices(parsedPlan.Resources, region)
	if prefetchErr := e.cache.PrewarmCache(ctx, requiredServices); prefetchErr != nil {
		log.WithError(prefetchErr).Warn("failed to prefetch some pricing data")
	}

	// Calculate costs for each resource
	for _, rc := range parsedPlan.Resources {
		attrs := rc.AfterValues
		if attrs == nil {
			attrs = rc.BeforeValues // delete has no after
		}

		req := ResolveRequest{
			ResourceType: rc.Type,
			Address:      rc.Address,
			Name:         rc.Name,
			ModuleAddr:   rc.ModuleAddr,
			Region:       region,
			Attrs:        attrs,
		}

		// Resolve resource cost including any compound sub-resources
		costs := e.resolver.ResolveWithSubResources(ctx, req)

		for i := range costs {
			// For the primary resource on update/replace, calculate before-state cost
			if i == 0 && (rc.Action == plan.ActionUpdate || rc.Action == plan.ActionReplace) && rc.BeforeValues != nil {
				e.resolver.ResolveBeforeCost(ctx, &costs[i], rc.Type, rc.BeforeValues, region)
			}

			result.Resources = append(result.Resources, costs[i])
			aggregateCost(result, costs[i], rc.Action)
		}
	}

	result.DiffCost = result.AfterCost - result.BeforeCost
	result.HasChanges = parsedPlan.HasChanges()
	result.Submodules = groupByModule(result.Resources)

	return result, nil
}

// aggregateCost adds a resource's cost to the module totals based on action.
func aggregateCost(result *ModuleCost, rc ResourceCost, action string) {
	if rc.IsUnsupported() {
		result.Unsupported++
		return
	}

	switch action {
	case plan.ActionCreate:
		result.AfterCost += rc.MonthlyCost
	case plan.ActionDelete:
		result.BeforeCost += rc.MonthlyCost
	case plan.ActionUpdate, plan.ActionReplace:
		result.BeforeCost += rc.BeforeMonthlyCost
		result.AfterCost += rc.MonthlyCost
	case plan.ActionNoOp:
		result.BeforeCost += rc.MonthlyCost
		result.AfterCost += rc.MonthlyCost
	}
}

// EstimateModules calculates costs for multiple modules concurrently.
func (e *Estimator) EstimateModules(ctx context.Context, modulePaths []string, regions map[string]string) (*EstimateResult, error) {
	const maxConcurrency = 4
	results := make([]ModuleCost, len(modulePaths))

	var g errgroup.Group
	g.SetLimit(maxConcurrency)

	for i, modulePath := range modulePaths {
		region := regions[modulePath]
		if region == "" {
			region = DefaultRegion
		}

		g.Go(func() error {
			mc, err := e.EstimateModule(ctx, modulePath, region)
			if err != nil {
				log.WithError(err).
					WithField("module", modulePath).
					Warn("failed to estimate module cost")
				results[i] = ModuleCost{
					ModuleID:   modulePath,
					ModulePath: modulePath,
					Error:      err.Error(),
				}
				return nil
			}
			results[i] = *mc
			return nil
		})
	}

	_ = g.Wait() //nolint:errcheck // individual errors collected in results

	result := &EstimateResult{
		Modules:     results,
		Currency:    "USD",
		GeneratedAt: time.Now().UTC(),
	}
	for i := range results {
		result.TotalBefore += results[i].BeforeCost
		result.TotalAfter += results[i].AfterCost
		if results[i].Error != "" {
			result.Errors = append(result.Errors, ModuleError{
				ModuleID: results[i].ModuleID,
				Error:    results[i].Error,
			})
		}
	}
	result.TotalDiff = result.TotalAfter - result.TotalBefore

	return result, nil
}

// ValidateAndPrefetch checks which pricing data is needed and downloads missing data.
func (e *Estimator) ValidateAndPrefetch(ctx context.Context, modulePaths []string, regions map[string]string) error {
	requiredServices := make(map[pricing.ServiceCode]map[string]bool)

	for _, modulePath := range modulePaths {
		planJSONPath := filepath.Join(modulePath, "plan.json")
		parsedPlan, err := plan.ParseJSON(planJSONPath)
		if err != nil {
			continue
		}

		region := regions[modulePath]
		if region == "" {
			region = DefaultRegion
		}

		for _, rc := range parsedPlan.Resources {
			h, ok := e.resolver.registry.GetHandler(rc.Type)
			if !ok || h.Category() != handler.CostCategoryStandard {
				continue
			}

			svc := h.ServiceCode()
			if requiredServices[svc] == nil {
				requiredServices[svc] = make(map[string]bool)
			}
			requiredServices[svc][region] = true
		}
	}

	services := make(map[pricing.ServiceCode][]string)
	for svc, regionMap := range requiredServices {
		for region := range regionMap {
			services[svc] = append(services[svc], region)
		}
	}

	if len(services) == 0 {
		log.Warn("no supported resources found in plans — nothing to price")
		return nil
	}

	log.WithField("services", len(services)).Debug("required pricing services")

	missing := e.cache.Validate(services)
	if len(missing) == 0 {
		log.Info("all pricing data is cached and valid")
		return nil
	}

	for i, m := range missing {
		log.WithField("service", string(m.Service)).
			WithField("region", m.Region).
			WithField("progress", fmt.Sprintf("%d/%d", i+1, len(missing))).
			Info("downloading pricing data")
		if _, err := e.cache.GetIndex(ctx, m.Service, m.Region); err != nil {
			return fmt.Errorf("fetch %s/%s pricing: %w", m.Service, m.Region, err)
		}
	}

	return nil
}
