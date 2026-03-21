package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

// EstimateModule calculates cost for a single module from plan.json and state.json
func (e *Estimator) EstimateModule(ctx context.Context, modulePath, region string) (*ModuleCost, error) {
	planJSONPath := filepath.Join(modulePath, "plan.json")
	stateJSONPath := filepath.Join(modulePath, "state.json")

	// Parse plan.json
	parsedPlan, err := plan.ParseJSON(planJSONPath)
	if err != nil {
		return nil, fmt.Errorf("parse plan.json: %w", err)
	}

	// Try to parse state.json for before costs
	var stateResources map[string]map[string]any
	if data, readErr := os.ReadFile(stateJSONPath); readErr == nil {
		stateResources = parseStateResources(data)
	}

	result := &ModuleCost{
		ModuleID:   strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath: modulePath,
		Region:     region,
		Resources:  make([]ResourceCost, 0),
	}

	// Collect required services for this module
	requiredServices := e.collectRequiredServices(parsedPlan.Resources, region)

	// Pre-fetch pricing data
	if err := e.prefetchPricing(ctx, requiredServices); err != nil {
		log.WithError(err).Warn("failed to prefetch some pricing data")
	}

	// Calculate costs for each resource change
	for _, rc := range parsedPlan.Resources {
		resourceCost := e.estimateResource(ctx, rc, region)

		// For update/replace, calculate before-state cost separately
		if rc.Action == "update" || rc.Action == "replace" {
			beforeAttrs := getBeforeAttrs(rc)
			if handler, ok := e.registry.GetHandler(rc.Type); ok && handler.Category() == aws.CostCategoryStandard {
				beforeResult := e.estimateStandardResource(ctx, handler, beforeAttrs, region, ResourceCost{})
				resourceCost.BeforeHourlyCost = beforeResult.HourlyCost
				resourceCost.BeforeMonthlyCost = beforeResult.MonthlyCost
			} else if ok && handler.Category() == aws.CostCategoryFixed {
				h, m := handler.CalculateCost(nil, beforeAttrs)
				resourceCost.BeforeHourlyCost = h
				resourceCost.BeforeMonthlyCost = m
			}
		}

		result.Resources = append(result.Resources, resourceCost)

		if resourceCost.IsUnsupported() {
			result.Unsupported++
			continue
		}

		// Aggregate before/after based on action
		switch rc.Action {
		case "create":
			result.AfterCost += resourceCost.MonthlyCost
		case "delete":
			result.BeforeCost += resourceCost.MonthlyCost
		case "update", "replace":
			result.BeforeCost += resourceCost.BeforeMonthlyCost
			result.AfterCost += resourceCost.MonthlyCost
		}
	}

	// Add costs from unchanged resources in state
	if stateResources != nil {
		unchangedCost := e.estimateUnchangedResources(ctx, parsedPlan, stateResources, region)
		result.BeforeCost += unchangedCost
		result.AfterCost += unchangedCost
	}

	result.DiffCost = result.AfterCost - result.BeforeCost
	result.HasChanges = result.DiffCost != 0 || len(parsedPlan.Resources) > 0

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

// estimateResource calculates cost for a single resource
func (e *Estimator) estimateResource(ctx context.Context, rc plan.ResourceChange, region string) ResourceCost {
	result := ResourceCost{
		Address: rc.Address,
		Type:    rc.Type,
		Name:    rc.Name,
		Region:  region,
	}

	handler, ok := e.registry.GetHandler(rc.Type)
	if !ok {
		result.ErrorKind = CostErrorNoHandler
		result.ErrorDetail = "no handler"
		aws.LogUnsupported(rc.Type, rc.Address)
		return result
	}

	attrs := getAfterAttrs(rc)

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

// estimateUnchangedResources calculates costs for resources in state that aren't changing
func (e *Estimator) estimateUnchangedResources(ctx context.Context, parsedPlan *plan.ParsedPlan, stateResources map[string]map[string]any, region string) float64 {
	// Build set of changed resource addresses
	changedAddrs := make(map[string]bool)
	for _, rc := range parsedPlan.Resources {
		changedAddrs[rc.Address] = true
	}

	var totalCost float64
	for addr, attrs := range stateResources {
		if changedAddrs[addr] {
			continue // Skip changed resources
		}

		// Extract resource type from address
		resourceType := extractResourceType(addr)
		if resourceType == "" {
			continue
		}

		handler, ok := e.registry.GetHandler(resourceType)
		if !ok {
			continue
		}

		lookup, err := handler.BuildLookup(region, attrs)
		if err != nil || lookup == nil {
			continue
		}

		index, err := e.cache.GetIndex(ctx, lookup.ServiceCode, region)
		if err != nil {
			continue
		}

		price, err := index.LookupPrice(*lookup)
		if err != nil {
			continue
		}

		_, monthly := handler.CalculateCost(price, attrs)
		totalCost += monthly
	}

	return totalCost
}

// getAfterAttrs extracts after-state attributes from a resource change.
func getAfterAttrs(rc plan.ResourceChange) map[string]any {
	attrs := make(map[string]any)
	for _, diff := range rc.Attributes {
		if diff.NewValue != "" && diff.NewValue != "(known after apply)" {
			attrs[diff.Path] = diff.NewValue
		} else if diff.OldValue != "" {
			attrs[diff.Path] = diff.OldValue
		}
	}
	return attrs
}

// getBeforeAttrs extracts before-state attributes from a resource change.
func getBeforeAttrs(rc plan.ResourceChange) map[string]any {
	attrs := make(map[string]any)
	for _, diff := range rc.Attributes {
		if diff.OldValue != "" {
			attrs[diff.Path] = diff.OldValue
		}
	}
	return attrs
}

// parseStateResources parses terraform state JSON to extract resource attributes
func parseStateResources(data []byte) map[string]map[string]any {
	var state struct {
		Resources []struct {
			Type      string `json:"type"`
			Name      string `json:"name"`
			Module    string `json:"module,omitempty"`
			Instances []struct {
				Attributes map[string]any `json:"attributes"`
				IndexKey   any            `json:"index_key,omitempty"`
			} `json:"instances"`
		} `json:"resources"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}

	result := make(map[string]map[string]any)
	for _, r := range state.Resources {
		for _, inst := range r.Instances {
			addr := buildResourceAddress(r.Module, r.Type, r.Name, inst.IndexKey)
			result[addr] = inst.Attributes
		}
	}

	return result
}

// buildResourceAddress constructs a resource address from components
func buildResourceAddress(module, resourceType, name string, indexKey any) string {
	var addr string
	if module != "" {
		addr = module + "."
	}
	addr += resourceType + "." + name

	if indexKey != nil {
		switch k := indexKey.(type) {
		case string:
			addr += fmt.Sprintf("[%q]", k)
		case float64:
			addr += fmt.Sprintf("[%d]", int(k))
		}
	}

	return addr
}

// extractResourceType extracts the resource type from an address
func extractResourceType(address string) string {
	// Remove module prefix if present
	parts := strings.Split(address, ".")
	for i, p := range parts {
		if p == "module" {
			continue
		}
		if strings.HasPrefix(p, "aws_") || strings.HasPrefix(p, "google_") || strings.HasPrefix(p, "azurerm_") {
			if i+1 < len(parts) {
				return p
			}
		}
	}
	return ""
}
