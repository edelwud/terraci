package cost

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
	"github.com/edelwud/terraci/internal/terraform/plan"
)

// Estimator calculates cost estimates for terraform plans
type Estimator struct {
	registry *aws.Registry
	cache    *pricing.Cache
}

// NewEstimator creates a new cost estimator
func NewEstimator(cacheDir string, cacheTTL time.Duration) *Estimator {
	return &Estimator{
		registry: aws.NewRegistry(),
		cache:    pricing.NewCache(cacheDir, cacheTTL),
	}
}

// CacheDir returns the resolved pricing cache directory path.
func (e *Estimator) CacheDir() string { return e.cache.Dir() }

// SetPricingFetcher replaces the pricing fetcher (for testing with httptest).
func (e *Estimator) SetPricingFetcher(f *pricing.Fetcher) { e.cache.SetFetcher(f) }

// CacheOldestAge returns the age of the oldest cache entry, or 0 if empty.
func (e *Estimator) CacheOldestAge() time.Duration { return e.cache.OldestAge() }

// CacheTTL returns the cache TTL.
func (e *Estimator) CacheTTL() time.Duration { return e.cache.TTL() }

// EstimateModule calculates cost for a single module from plan.json.
// plan.json contains all resources (including unchanged no-op) with full before/after state.
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

	// Collect required services and pre-fetch pricing
	requiredServices := e.collectRequiredServices(parsedPlan.Resources, region)
	if err := e.prefetchPricing(ctx, requiredServices); err != nil {
		log.WithError(err).Warn("failed to prefetch some pricing data")
	}

	// Calculate costs for each resource (including no-op unchanged resources)
	for _, rc := range parsedPlan.Resources {
		// Use full attribute maps from plan JSON (not diff-based)
		attrs := rc.AfterValues
		if attrs == nil {
			attrs = rc.BeforeValues // delete has no after
		}

		resourceCost := e.estimateResourceFromAttrs(ctx, rc.Type, rc.Address, rc.Name, region, attrs)

		// For update/replace, calculate before-state cost separately
		if rc.Action == plan.ActionUpdate || rc.Action == plan.ActionReplace {
			beforeAttrs := rc.BeforeValues
			if handler, ok := e.registry.GetHandler(rc.Type); ok && beforeAttrs != nil {
				switch handler.Category() {
				case aws.CostCategoryStandard:
					beforeResult := e.estimateStandardResource(ctx, handler, beforeAttrs, region, ResourceCost{})
					resourceCost.BeforeHourlyCost = beforeResult.HourlyCost
					resourceCost.BeforeMonthlyCost = beforeResult.MonthlyCost
				case aws.CostCategoryFixed:
					h, m := handler.CalculateCost(nil, beforeAttrs)
					resourceCost.BeforeHourlyCost = h
					resourceCost.BeforeMonthlyCost = m
				case aws.CostCategoryUsageBased:
					// no cost
				}
			}
		}

		result.Resources = append(result.Resources, resourceCost)

		if resourceCost.IsUnsupported() {
			result.Unsupported++
			continue
		}

		// Aggregate before/after based on action
		switch rc.Action {
		case plan.ActionCreate:
			result.AfterCost += resourceCost.MonthlyCost
		case plan.ActionDelete:
			result.BeforeCost += resourceCost.MonthlyCost
		case plan.ActionUpdate, plan.ActionReplace:
			result.BeforeCost += resourceCost.BeforeMonthlyCost
			result.AfterCost += resourceCost.MonthlyCost
		case plan.ActionNoOp:
			result.BeforeCost += resourceCost.MonthlyCost
			result.AfterCost += resourceCost.MonthlyCost
		}
	}

	result.DiffCost = result.AfterCost - result.BeforeCost
	result.HasChanges = parsedPlan.HasChanges()

	return result, nil
}

// EstimateModules calculates costs for multiple modules concurrently.
func (e *Estimator) EstimateModules(ctx context.Context, modulePaths []string, regions map[string]string) (*EstimateResult, error) {
	results := make([]ModuleCost, len(modulePaths))

	var g errgroup.Group
	g.SetLimit(4)

	for i, modulePath := range modulePaths {
		region := regions[modulePath]
		if region == "" {
			region = "us-east-1"
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
				return nil // collect per-module errors, don't fail the group
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

// ValidateAndPrefetch checks which pricing data is needed and downloads missing data
func (e *Estimator) ValidateAndPrefetch(ctx context.Context, modulePaths []string, regions map[string]string) error {
	// Scan all modules to determine required services
	requiredServices := make(map[pricing.ServiceCode]map[string]bool)

	for _, modulePath := range modulePaths {
		planJSONPath := filepath.Join(modulePath, "plan.json")
		parsedPlan, err := plan.ParseJSON(planJSONPath)
		if err != nil {
			continue // Skip modules without valid plan.json
		}

		region := regions[modulePath]
		if region == "" {
			region = "us-east-1"
		}

		for _, rc := range parsedPlan.Resources {
			handler, ok := e.registry.GetHandler(rc.Type)
			if !ok || handler.Category() != aws.CostCategoryStandard {
				continue
			}

			svc := handler.ServiceCode()
			if requiredServices[svc] == nil {
				requiredServices[svc] = make(map[string]bool)
			}
			requiredServices[svc][region] = true
		}
	}

	// Convert to format expected by cache
	services := make(map[pricing.ServiceCode][]string)
	for svc, regionMap := range requiredServices {
		for region := range regionMap {
			services[svc] = append(services[svc], region)
		}
	}

	// Check what's missing
	missing := e.cache.Validate(services)
	if len(missing) == 0 {
		log.Debug("all required pricing data is cached")
		return nil
	}

	log.WithField("count", len(missing)).Info("downloading missing pricing data")

	// Download missing data
	for _, m := range missing {
		if _, err := e.cache.GetIndex(ctx, m.Service, m.Region); err != nil {
			return fmt.Errorf("fetch %s/%s pricing: %w", m.Service, m.Region, err)
		}
	}

	return nil
}

// estimateResourceFromAttrs calculates cost for a resource given its full attributes.
func (e *Estimator) estimateResourceFromAttrs(ctx context.Context, resourceType, address, name, region string, attrs map[string]any) ResourceCost {
	result := ResourceCost{
		Address: address,
		Type:    resourceType,
		Name:    name,
		Region:  region,
	}

	handler, ok := e.registry.GetHandler(resourceType)
	if !ok {
		result.ErrorKind = CostErrorNoHandler
		result.ErrorDetail = "no handler"
		aws.LogUnsupported(resourceType, address)
		return result
	}

	if attrs == nil {
		attrs = make(map[string]any)
	}

	// Dispatch by handler category
	switch handler.Category() {
	case aws.CostCategoryUsageBased:
		result.ErrorKind = CostErrorUsageBased
		result.ErrorDetail = "usage-based"
		result.PriceSource = "usage-based"
		return result

	case aws.CostCategoryFixed:
		hourly, monthly := handler.CalculateCost(nil, attrs)
		result.HourlyCost = hourly
		result.MonthlyCost = monthly
		result.PriceSource = "fixed"
		return result

	case aws.CostCategoryStandard:
		return e.estimateStandardResource(ctx, handler, attrs, region, result)
	}

	return result
}

// estimateStandardResource handles the full pricing API lookup path.
func (e *Estimator) estimateStandardResource(ctx context.Context, handler aws.ResourceHandler, attrs map[string]any, region string, result ResourceCost) ResourceCost {
	lookup, err := handler.BuildLookup(region, attrs)
	if err != nil {
		result.ErrorKind = CostErrorLookupFailed
		result.ErrorDetail = err.Error()
		return result
	}

	if lookup == nil {
		return result
	}

	index, err := e.cache.GetIndex(ctx, lookup.ServiceCode, region)
	if err != nil {
		log.WithError(err).
			WithField("service", lookup.ServiceCode).
			WithField("region", region).
			Debug("failed to get pricing index")
		result.ErrorKind = CostErrorAPIFailure
		result.ErrorDetail = "pricing unavailable"
		return result
	}

	price, err := index.LookupPrice(*lookup)
	if err != nil {
		log.WithError(err).
			WithField("address", result.Address).
			Debug("price lookup failed")
		result.ErrorKind = CostErrorNoPrice
		result.ErrorDetail = "no matching price"
		return result
	}

	hourly, monthly := handler.CalculateCost(price, attrs)
	result.HourlyCost = hourly
	result.MonthlyCost = monthly
	result.PriceSource = "aws-bulk-api"

	return result
}

// collectRequiredServices determines which AWS services need pricing data
func (e *Estimator) collectRequiredServices(resources []plan.ResourceChange, region string) map[pricing.ServiceCode][]string {
	services := make(map[pricing.ServiceCode]map[string]bool)

	for _, rc := range resources {
		handler, ok := e.registry.GetHandler(rc.Type)
		if !ok || handler.Category() != aws.CostCategoryStandard {
			continue
		}

		svc := handler.ServiceCode()
		if services[svc] == nil {
			services[svc] = make(map[string]bool)
		}
		services[svc][region] = true
	}

	// Convert to slice format
	result := make(map[pricing.ServiceCode][]string)
	for svc, regionMap := range services {
		for r := range regionMap {
			result[svc] = append(result[svc], r)
		}
	}

	return result
}

// prefetchPricing downloads pricing data for required services
func (e *Estimator) prefetchPricing(ctx context.Context, services map[pricing.ServiceCode][]string) error {
	return e.cache.PrewarmCache(ctx, services)
}
