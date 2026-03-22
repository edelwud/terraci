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

// DefaultRegion is used when no region is specified.
const DefaultRegion = "us-east-1"

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

// CleanExpiredCache removes expired cache entries.
func (e *Estimator) CleanExpiredCache() {
	if err := e.cache.CleanExpired(); err != nil {
		log.WithError(err).Debug("failed to clean expired cache")
	}
}

// CacheEntries returns info about all cached pricing files.
func (e *Estimator) CacheEntries() []pricing.CacheEntry { return e.cache.Entries() }

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

		resourceCost := e.estimateResourceFromAttrs(ctx, rc.Type, rc.Address, rc.Name, rc.ModuleAddr, region, attrs)

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
					h, m := handler.CalculateCost(nil, nil, region, beforeAttrs)
					resourceCost.BeforeHourlyCost = h
					resourceCost.BeforeMonthlyCost = m
				case aws.CostCategoryUsageBased:
					// no cost
				}
			}
		}

		result.Resources = append(result.Resources, resourceCost)
		e.aggregateCost(result, resourceCost, rc.Action)

		// Synthesize sub-resources from CompoundHandler (e.g., root_block_device → EBS)
		if handler, ok := e.registry.GetHandler(rc.Type); ok {
			if ch, ok := handler.(aws.CompoundHandler); ok {
				for _, sub := range ch.SubResources(attrs) {
					subAddr := rc.Address + sub.Suffix
					subCost := e.estimateResourceFromAttrs(ctx, sub.Type, subAddr, sub.Suffix, rc.ModuleAddr, region, sub.Attrs)
					result.Resources = append(result.Resources, subCost)
					e.aggregateCost(result, subCost, rc.Action)
				}
			}
		}
	}

	result.DiffCost = result.AfterCost - result.BeforeCost
	result.HasChanges = parsedPlan.HasChanges()
	result.Submodules = groupByModule(result.Resources)

	return result, nil
}

// aggregateCost adds a resource's cost to the module totals based on action.
func (e *Estimator) aggregateCost(result *ModuleCost, rc ResourceCost, action string) {
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

// groupByModule groups resources by their Terraform module address into a tree.
// Child modules (e.g., module.eks.module.node_group) are nested under parents (module.eks).
func groupByModule(resources []ResourceCost) []SubmoduleCost {
	// Step 1: group resources by exact ModuleAddr
	type flatGroup struct {
		addr      string
		resources []ResourceCost
		cost      float64
	}
	groups := make(map[string]*flatGroup)
	var order []string

	for i := range resources {
		rc := &resources[i]
		addr := rc.ModuleAddr
		if groups[addr] == nil {
			groups[addr] = &flatGroup{addr: addr}
			order = append(order, addr)
		}
		groups[addr].resources = append(groups[addr].resources, *rc)
		groups[addr].cost += rc.MonthlyCost
	}

	// Step 2: build tree — find parent for each address
	// "module.eks.module.node_group" is a child of "module.eks"
	nodes := make(map[string]*SubmoduleCost, len(order))
	for _, addr := range order {
		g := groups[addr]
		nodes[addr] = &SubmoduleCost{
			ModuleAddr:  addr,
			MonthlyCost: g.cost,
			Resources:   g.resources,
		}
	}

	// Step 3: attach children to parents, collect roots
	var roots []SubmoduleCost
	attached := make(map[string]bool)

	for _, addr := range order {
		parent := findParentAddr(addr, nodes)
		if parent != "" {
			nodes[parent].Children = append(nodes[parent].Children, *nodes[addr])
			nodes[parent].MonthlyCost += nodes[addr].MonthlyCost
			attached[addr] = true
		}
	}

	for _, addr := range order {
		if !attached[addr] {
			roots = append(roots, *nodes[addr])
		}
	}

	return roots
}

// findParentAddr finds the nearest existing parent module address.
// For "module.eks.module.node_group", checks "module.eks" in the nodes map.
func findParentAddr(addr string, nodes map[string]*SubmoduleCost) string {
	// Walk backwards through "module.X.module.Y" to find "module.X"
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == '.' {
			// Check if the prefix up to this dot is a "module.X" boundary
			candidate := addr[:i]
			// Must end at a module boundary: "module.eks" not "module.ek"
			// The part after candidate + "." should start with "module."
			rest := addr[len(candidate)+1:]
			if len(rest) >= 7 && rest[:7] == "module." {
				if _, ok := nodes[candidate]; ok {
					return candidate
				}
			}
		}
	}
	return ""
}

// EstimateModules calculates costs for multiple modules concurrently.
func (e *Estimator) EstimateModules(ctx context.Context, modulePaths []string, regions map[string]string) (*EstimateResult, error) {
	results := make([]ModuleCost, len(modulePaths))

	var g errgroup.Group
	g.SetLimit(4)

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
			region = DefaultRegion
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

	if len(services) == 0 {
		log.Warn("no supported resources found in plans — nothing to price")
		return nil
	}

	log.WithField("services", len(services)).Debug("required pricing services")
	for svc, regions := range services {
		log.WithField("service", string(svc)).WithField("regions", regions).Debug("need pricing for")
	}

	// Check what's missing
	missing := e.cache.Validate(services)
	if len(missing) == 0 {
		log.Info("all pricing data is cached and valid")
		return nil
	}

	// Download missing/expired data
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

// estimateResourceFromAttrs calculates cost for a resource given its full attributes.
func (e *Estimator) estimateResourceFromAttrs(ctx context.Context, resourceType, address, name, moduleAddr, region string, attrs map[string]any) ResourceCost {
	result := ResourceCost{
		Address:    address,
		ModuleAddr: moduleAddr,
		Type:       resourceType,
		Name:       name,
		Region:     region,
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

	result.Details = handler.Describe(nil, attrs)

	// Dispatch by handler category
	switch handler.Category() {
	case aws.CostCategoryUsageBased:
		result.ErrorKind = CostErrorUsageBased
		result.ErrorDetail = "usage-based"
		result.PriceSource = "usage-based"
		return result

	case aws.CostCategoryFixed:
		hourly, monthly := handler.CalculateCost(nil, nil, region, attrs)
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

	hourly, monthly := handler.CalculateCost(price, index, region, attrs)
	result.HourlyCost = hourly
	result.MonthlyCost = monthly
	result.PriceSource = "aws-bulk-api"
	result.Details = handler.Describe(price, attrs)

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
