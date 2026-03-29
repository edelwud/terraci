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
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/provider"
)

// DefaultRegion is used when no region is specified.
const DefaultRegion = "us-east-1"

// Estimator calculates cost estimates for terraform plans.
type Estimator struct {
	registry *provider.Registry
	cache    *pricing.Cache
}

// NewEstimator creates a new cost estimator with the given pricing fetcher.
func NewEstimator(cacheDir string, cacheTTL time.Duration, fetcher pricing.PriceFetcher) *Estimator {
	return &Estimator{
		registry: newDefaultRegistry(),
		cache:    pricing.NewCache(cacheDir, cacheTTL, fetcher),
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
	requiredServices := e.collectRequiredServices(parsedPlan.Resources, region)
	if prefetchErr := e.cache.PrewarmCache(ctx, requiredServices); prefetchErr != nil {
		log.WithError(prefetchErr).Warn("failed to prefetch some pricing data")
	}

	// Calculate costs for each resource
	for _, rc := range parsedPlan.Resources {
		attrs := rc.AfterValues
		if attrs == nil {
			attrs = rc.BeforeValues // delete has no after
		}

		resourceCost := e.resolveResourceCost(ctx, rc.Type, rc.Address, rc.Name, rc.ModuleAddr, region, attrs)

		// For update/replace, calculate before-state cost separately
		if (rc.Action == plan.ActionUpdate || rc.Action == plan.ActionReplace) && rc.BeforeValues != nil {
			e.resolveBeforeCost(ctx, &resourceCost, rc.Type, rc.BeforeValues, region)
		}

		result.Resources = append(result.Resources, resourceCost)
		aggregateCost(result, resourceCost, rc.Action)

		// Synthesize sub-resources from CompoundHandler (e.g., root_block_device → EBS)
		e.synthesizeSubResources(ctx, result, rc, attrs, region)
	}

	result.DiffCost = result.AfterCost - result.BeforeCost
	result.HasChanges = parsedPlan.HasChanges()
	result.Submodules = groupByModule(result.Resources)

	return result, nil
}

// resolveBeforeCost calculates the before-state cost for update/replace resources.
func (e *Estimator) resolveBeforeCost(ctx context.Context, rc *ResourceCost, resourceType string, beforeAttrs map[string]any, region string) {
	handler, ok := e.registry.GetHandler(resourceType)
	if !ok {
		return
	}

	switch handler.Category() {
	case provider.CostCategoryStandard:
		before := e.resolveStandardCost(ctx, handler, beforeAttrs, region, ResourceCost{})
		rc.BeforeHourlyCost = before.HourlyCost
		rc.BeforeMonthlyCost = before.MonthlyCost
	case provider.CostCategoryFixed:
		h, m := handler.CalculateCost(nil, nil, region, beforeAttrs)
		rc.BeforeHourlyCost = h
		rc.BeforeMonthlyCost = m
	case provider.CostCategoryUsageBased:
		// no cost
	}
}

// synthesizeSubResources creates cost entries for compound resources (e.g., EC2 root_block_device).
func (e *Estimator) synthesizeSubResources(ctx context.Context, result *ModuleCost, rc plan.ResourceChange, attrs map[string]any, region string) {
	handler, ok := e.registry.GetHandler(rc.Type)
	if !ok {
		return
	}

	ch, ok := handler.(provider.CompoundHandler)
	if !ok {
		return
	}

	for _, sub := range ch.SubResources(attrs) {
		subAddr := rc.Address + sub.Suffix
		subCost := e.resolveResourceCost(ctx, sub.Type, subAddr, sub.Suffix, rc.ModuleAddr, region, sub.Attrs)
		result.Resources = append(result.Resources, subCost)
		aggregateCost(result, subCost, rc.Action)
	}
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
			handler, ok := e.registry.GetHandler(rc.Type)
			if !ok || handler.Category() != provider.CostCategoryStandard {
				continue
			}

			svc := handler.ServiceCode()
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
