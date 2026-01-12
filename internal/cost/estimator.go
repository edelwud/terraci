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
	var stateResources map[string]map[string]interface{}
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
		resourceCost := e.estimateResource(ctx, rc, region, stateResources)
		result.Resources = append(result.Resources, resourceCost)

		if resourceCost.Unsupported {
			result.Unsupported++
			continue
		}

		// Calculate before/after based on action
		switch rc.Action {
		case "create":
			result.AfterCost += resourceCost.MonthlyCost
		case "delete":
			result.BeforeCost += resourceCost.MonthlyCost
		case "update", "replace":
			// For updates, we need before and after values
			// Current implementation uses after cost for both (simplified)
			result.BeforeCost += resourceCost.MonthlyCost
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

// EstimateModules calculates costs for multiple modules
func (e *Estimator) EstimateModules(ctx context.Context, modulePaths []string, regions map[string]string) (*EstimateResult, error) {
	result := &EstimateResult{
		Modules:     make([]ModuleCost, 0, len(modulePaths)),
		Currency:    "USD",
		GeneratedAt: time.Now().UTC(),
	}

	for _, modulePath := range modulePaths {
		region := regions[modulePath]
		if region == "" {
			region = "us-east-1" // Default
		}

		moduleCost, err := e.EstimateModule(ctx, modulePath, region)
		if err != nil {
			log.WithError(err).
				WithField("module", modulePath).
				Warn("failed to estimate module cost")
			result.Modules = append(result.Modules, ModuleCost{
				ModuleID:   modulePath,
				ModulePath: modulePath,
				Error:      err.Error(),
			})
			continue
		}

		result.Modules = append(result.Modules, *moduleCost)
		result.TotalBefore += moduleCost.BeforeCost
		result.TotalAfter += moduleCost.AfterCost
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
			if !ok {
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
func (e *Estimator) estimateResource(ctx context.Context, rc plan.ResourceChange, region string, _ map[string]map[string]interface{}) ResourceCost {
	result := ResourceCost{
		Address: rc.Address,
		Type:    rc.Type,
		Name:    rc.Name,
		Region:  region,
	}

	handler, ok := e.registry.GetHandler(rc.Type)
	if !ok {
		result.Unsupported = true
		result.UnsupportedBy = "no handler"
		aws.LogUnsupported(rc.Type, rc.Address)
		return result
	}

	// Get resource attributes (from plan's after state)
	attrs := getResourceAttrs(rc)

	// Build pricing lookup
	lookup, err := handler.BuildLookup(region, attrs)
	if err != nil {
		result.Unsupported = true
		result.UnsupportedBy = err.Error()
		return result
	}

	if lookup == nil {
		// Handler explicitly returned nil (usage-based pricing)
		return result
	}

	// Get price from cache
	index, err := e.cache.GetIndex(ctx, lookup.ServiceCode, region)
	if err != nil {
		log.WithError(err).
			WithField("service", lookup.ServiceCode).
			WithField("region", region).
			Debug("failed to get pricing index")
		result.Unsupported = true
		result.UnsupportedBy = "pricing unavailable"
		return result
	}

	price, err := index.LookupPrice(*lookup)
	if err != nil {
		log.WithError(err).
			WithField("address", rc.Address).
			Debug("price lookup failed")
		result.Unsupported = true
		result.UnsupportedBy = "no matching price"
		return result
	}

	// Calculate cost
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
		if !ok {
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
func (e *Estimator) estimateUnchangedResources(ctx context.Context, parsedPlan *plan.ParsedPlan, stateResources map[string]map[string]interface{}, region string) float64 {
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

// getResourceAttrs extracts attributes from a resource change
func getResourceAttrs(rc plan.ResourceChange) map[string]interface{} {
	// Use attributes from the parsed plan
	// In a full implementation, this would parse the plan JSON directly
	// For now, return empty map - handlers use defaults
	attrs := make(map[string]interface{})

	// Extract from AttrDiff if available
	for _, diff := range rc.Attributes {
		if diff.NewValue != "" && diff.NewValue != "(known after apply)" {
			attrs[diff.Path] = diff.NewValue
		} else if diff.OldValue != "" {
			attrs[diff.Path] = diff.OldValue
		}
	}

	return attrs
}

// parseStateResources parses terraform state JSON to extract resource attributes
func parseStateResources(data []byte) map[string]map[string]interface{} {
	var state struct {
		Resources []struct {
			Type      string `json:"type"`
			Name      string `json:"name"`
			Module    string `json:"module,omitempty"`
			Instances []struct {
				Attributes map[string]interface{} `json:"attributes"`
				IndexKey   interface{}            `json:"index_key,omitempty"`
			} `json:"instances"`
		} `json:"resources"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}

	result := make(map[string]map[string]interface{})
	for _, r := range state.Resources {
		for _, inst := range r.Instances {
			addr := buildResourceAddress(r.Module, r.Type, r.Name, inst.IndexKey)
			result[addr] = inst.Attributes
		}
	}

	return result
}

// buildResourceAddress constructs a resource address from components
func buildResourceAddress(module, resourceType, name string, indexKey interface{}) string {
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
