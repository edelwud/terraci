package costengine

import (
	"context"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const priceSourceUsageBased = "usage-based"

// resourceChange is a type alias to avoid leaking internal/terraform/plan into the resolver interface.
type resourceChange = plan.ResourceChange

// CostResolver handles the cost resolution logic: looking up handlers,
// fetching pricing, and calculating costs. Decoupled from the Estimator
// orchestration layer.
//
// Supports middleware for intercepting cost resolution (discounts, overrides, logging).
type CostResolver struct {
	registry   RegistryLookup
	pricing    PricingSource
	middleware []CostMiddleware
}

// NewCostResolver creates a new resolver with the given registry and pricing source.
func NewCostResolver(registry RegistryLookup, pricingSrc PricingSource) *CostResolver {
	return &CostResolver{
		registry: registry,
		pricing:  pricingSrc,
	}
}

// Use appends a middleware to the resolver chain.
// Middleware is applied in order: the first added is the outermost wrapper.
func (r *CostResolver) Use(mw CostMiddleware) {
	r.middleware = append(r.middleware, mw)
}

// Resolve calculates cost for a single resource, applying any registered middleware.
func (r *CostResolver) Resolve(ctx context.Context, req ResolveRequest) ResourceCost {
	if len(r.middleware) > 0 {
		return r.resolveWithMiddleware(ctx, req)
	}
	return r.coreResolve(ctx, req)
}

// resolveWithMiddleware builds the middleware chain and executes it.
func (r *CostResolver) resolveWithMiddleware(ctx context.Context, req ResolveRequest) ResourceCost {
	fn := r.coreResolve
	for i := len(r.middleware) - 1; i >= 0; i-- {
		mw := r.middleware[i]
		next := fn
		fn = func(ctx context.Context, req ResolveRequest) ResourceCost {
			return mw(ctx, next, req)
		}
	}
	return fn(ctx, req)
}

// coreResolve is the actual resolution logic without middleware.
func (r *CostResolver) coreResolve(ctx context.Context, req ResolveRequest) ResourceCost {
	result := ResourceCost{
		Address:    req.Address,
		ModuleAddr: req.ModuleAddr,
		Type:       req.ResourceType,
		Name:       req.Name,
		Region:     req.Region,
	}

	h, ok := r.registry.GetHandler(req.ResourceType)
	if !ok {
		result.ErrorKind = CostErrorNoHandler
		result.ErrorDetail = "no handler"
		handler.LogUnsupported(req.ResourceType, req.Address)
		return result
	}

	attrs := req.Attrs
	if attrs == nil {
		attrs = make(map[string]any)
	}

	result.Details = h.Describe(nil, attrs)

	switch h.Category() {
	case handler.CostCategoryUsageBased:
		result.ErrorKind = CostErrorUsageBased
		result.ErrorDetail = priceSourceUsageBased
		result.PriceSource = priceSourceUsageBased
		return result

	case handler.CostCategoryFixed:
		hourly, monthly := h.CalculateCost(nil, nil, req.Region, attrs)
		result.HourlyCost = hourly
		result.MonthlyCost = monthly
		result.PriceSource = "fixed"
		return result

	case handler.CostCategoryStandard:
		return r.resolveStandardCost(ctx, h, attrs, req.Region, result)
	}

	return result
}

// ResolveBeforeCost calculates the before-state cost for update/replace resources.
func (r *CostResolver) ResolveBeforeCost(ctx context.Context, rc *ResourceCost, resourceType string, beforeAttrs map[string]any, region string) {
	h, ok := r.registry.GetHandler(resourceType)
	if !ok {
		return
	}

	switch h.Category() {
	case handler.CostCategoryStandard:
		before := r.resolveStandardCost(ctx, h, beforeAttrs, region, ResourceCost{})
		rc.BeforeHourlyCost = before.HourlyCost
		rc.BeforeMonthlyCost = before.MonthlyCost
	case handler.CostCategoryFixed:
		hourly, monthly := h.CalculateCost(nil, nil, region, beforeAttrs)
		rc.BeforeHourlyCost = hourly
		rc.BeforeMonthlyCost = monthly
	case handler.CostCategoryUsageBased:
		// no cost
	}
}

// ResolveWithSubResources resolves a resource and any compound sub-resources
// (e.g., EC2 root_block_device -> EBS volume).
func (r *CostResolver) ResolveWithSubResources(ctx context.Context, req ResolveRequest) []ResourceCost {
	primary := r.Resolve(ctx, req)
	results := []ResourceCost{primary}

	h, ok := r.registry.GetHandler(req.ResourceType)
	if !ok {
		return results
	}

	ch, ok := h.(handler.CompoundHandler)
	if !ok {
		return results
	}

	for _, sub := range ch.SubResources(req.Attrs) {
		subReq := ResolveRequest{
			ResourceType: sub.Type,
			Address:      req.Address + sub.Suffix,
			Name:         sub.Suffix,
			ModuleAddr:   req.ModuleAddr,
			Region:       req.Region,
			Attrs:        sub.Attrs,
		}
		results = append(results, r.Resolve(ctx, subReq))
	}

	return results
}

// resolveStandardCost handles the full pricing API lookup path.
func (r *CostResolver) resolveStandardCost(ctx context.Context, h handler.ResourceHandler, attrs map[string]any, region string, result ResourceCost) ResourceCost {
	lookup, err := h.BuildLookup(region, attrs)
	if err != nil {
		result.ErrorKind = CostErrorLookupFailed
		result.ErrorDetail = err.Error()
		return result
	}

	if lookup == nil {
		return result
	}

	index, err := r.pricing.GetIndex(ctx, lookup.ServiceCode, region)
	if err != nil {
		log.WithError(err).
			WithField("service", lookup.ServiceCode).
			WithField("region", region).
			Debug("failed to get pricing index")
		result.ErrorKind = CostErrorAPIFailure
		result.ErrorDetail = "pricing unavailable"
		return result
	}

	if index == nil {
		result.ErrorKind = CostErrorAPIFailure
		result.ErrorDetail = "empty pricing index"
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

	hourly, monthly := h.CalculateCost(price, index, region, attrs)
	result.HourlyCost = hourly
	result.MonthlyCost = monthly
	result.PriceSource = "aws-bulk-api"
	result.Details = h.Describe(price, attrs)

	return result
}

// collectRequiredServices determines which services need pricing data.
func (r *CostResolver) collectRequiredServices(resources []resourceChange, region string) map[pricing.ServiceCode][]string {
	services := make(map[pricing.ServiceCode]map[string]bool)

	for _, rc := range resources {
		h, ok := r.registry.GetHandler(rc.Type)
		if !ok || h.Category() != handler.CostCategoryStandard {
			continue
		}

		svc := h.ServiceCode()
		if services[svc] == nil {
			services[svc] = make(map[string]bool)
		}
		services[svc][region] = true
	}

	result := make(map[pricing.ServiceCode][]string)
	for svc, regionMap := range services {
		for r := range regionMap {
			result[svc] = append(result[svc], r)
		}
	}

	return result
}
