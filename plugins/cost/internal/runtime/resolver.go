package runtime

import (
	"context"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const priceSourceUsageBased = "usage-based"

// RegistryLookup is the minimal interface for finding resource handlers.
type RegistryLookup interface {
	ResolveHandler(providerID string, resourceType handler.ResourceType) (handler.ResourceHandler, bool)
}

// PricingSource abstracts pricing index retrieval for cost resolution.
type PricingSource interface {
	GetIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error)
	SourceName(providerID string) string
}

// ResolveRequest bundles all inputs for a single resource cost resolution.
type ResolveRequest struct {
	ResourceType handler.ResourceType
	Address      string
	Name         string
	ModuleAddr   string
	Region       string
	Attrs        map[string]any
}

// ResolveFunc is the signature of a cost resolution function.
type ResolveFunc func(ctx context.Context, req ResolveRequest) model.ResourceCost

// CostMiddleware wraps a cost resolution step.
type CostMiddleware func(ctx context.Context, next ResolveFunc, req ResolveRequest) model.ResourceCost

// CostResolver handles the cost resolution logic.
type CostResolver struct {
	router     ProviderRouter
	registry   RegistryLookup
	pricing    PricingSource
	middleware []CostMiddleware
}

// NewCostResolver creates a new resolver with the given registry and pricing source.
func NewCostResolver(router ProviderRouter, registry RegistryLookup, pricingSrc PricingSource) *CostResolver {
	return &CostResolver{
		router:   router,
		registry: registry,
		pricing:  pricingSrc,
	}
}

// Registry returns the underlying handler lookup dependency.
func (r *CostResolver) Registry() RegistryLookup {
	return r.registry
}

// Use appends a middleware to the resolver chain.
func (r *CostResolver) Use(mw CostMiddleware) {
	r.middleware = append(r.middleware, mw)
}

// Resolve calculates cost for a single resource.
func (r *CostResolver) Resolve(ctx context.Context, req ResolveRequest) model.ResourceCost {
	if len(r.middleware) > 0 {
		return r.resolveWithMiddleware(ctx, req)
	}
	return r.coreResolve(ctx, req)
}

func (r *CostResolver) resolveWithMiddleware(ctx context.Context, req ResolveRequest) model.ResourceCost {
	fn := r.coreResolve
	for i := len(r.middleware) - 1; i >= 0; i-- {
		mw := r.middleware[i]
		next := fn
		fn = func(ctx context.Context, req ResolveRequest) model.ResourceCost {
			return mw(ctx, next, req)
		}
	}
	return fn(ctx, req)
}

func (r *CostResolver) coreResolve(ctx context.Context, req ResolveRequest) model.ResourceCost {
	result := model.ResourceCost{
		Provider:   "",
		Address:    req.Address,
		ModuleAddr: req.ModuleAddr,
		Type:       req.ResourceType.String(),
		Name:       req.Name,
		Region:     req.Region,
	}

	providerID, ok := r.router.ResolveProvider(req.ResourceType)
	if !ok {
		result.ErrorKind = model.CostErrorNoProvider
		result.ErrorDetail = "no provider"
		handler.LogUnsupported(req.ResourceType.String(), req.Address)
		return result
	}
	result.Provider = providerID

	h, ok := r.registry.ResolveHandler(providerID, req.ResourceType)
	if !ok {
		result.ErrorKind = model.CostErrorNoHandler
		result.ErrorDetail = "no handler"
		return result
	}

	attrs := req.Attrs
	if attrs == nil {
		attrs = make(map[string]any)
	}

	result.Details = describeResource(h, nil, attrs)

	switch h.Category() {
	case handler.CostCategoryUsageBased:
		result.ErrorKind = model.CostErrorUsageBased
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
		return r.resolveStandardCost(ctx, providerID, h, attrs, req.Region, result)
	default:
		return result
	}
}

// ResolveBeforeCost calculates the before-state cost for update/replace resources.
func (r *CostResolver) ResolveBeforeCost(ctx context.Context, rc *model.ResourceCost, resourceType handler.ResourceType, beforeAttrs map[string]any, region string) {
	providerID, ok := r.router.ResolveProvider(resourceType)
	if !ok {
		return
	}
	h, ok := r.registry.ResolveHandler(providerID, resourceType)
	if !ok {
		return
	}

	switch h.Category() {
	case handler.CostCategoryStandard:
		before := r.resolveStandardCost(ctx, providerID, h, beforeAttrs, region, model.ResourceCost{Provider: providerID})
		rc.BeforeHourlyCost = before.HourlyCost
		rc.BeforeMonthlyCost = before.MonthlyCost
	case handler.CostCategoryFixed:
		hourly, monthly := h.CalculateCost(nil, nil, region, beforeAttrs)
		rc.BeforeHourlyCost = hourly
		rc.BeforeMonthlyCost = monthly
	case handler.CostCategoryUsageBased:
	}
}

// ResolveWithSubResources resolves a resource and any compound sub-resources.
func (r *CostResolver) ResolveWithSubResources(ctx context.Context, req ResolveRequest) []model.ResourceCost {
	primary := r.Resolve(ctx, req)
	results := []model.ResourceCost{primary}

	providerID, ok := r.router.ResolveProvider(req.ResourceType)
	if !ok {
		return results
	}

	h, ok := r.registry.ResolveHandler(providerID, req.ResourceType)
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

func (r *CostResolver) resolveStandardCost(ctx context.Context, providerID string, h handler.ResourceHandler, attrs map[string]any, region string, result model.ResourceCost) model.ResourceCost {
	lookupBuilder, ok := h.(handler.LookupBuilder)
	if !ok {
		result.ErrorKind = model.CostErrorLookupFailed
		result.ErrorDetail = "lookup builder not implemented"
		return result
	}

	lookup, err := lookupBuilder.BuildLookup(region, attrs)
	if err != nil {
		result.ErrorKind = model.CostErrorLookupFailed
		result.ErrorDetail = err.Error()
		return result
	}
	if lookup == nil {
		return result
	}

	index, err := r.pricing.GetIndex(ctx, lookup.ServiceID, region)
	if err != nil {
		log.WithError(err).WithField("service", lookup.ServiceID.String()).WithField("region", region).Debug("failed to get pricing index")
		result.ErrorKind = model.CostErrorAPIFailure
		result.ErrorDetail = "pricing unavailable"
		return result
	}
	if index == nil {
		result.ErrorKind = model.CostErrorAPIFailure
		result.ErrorDetail = "empty pricing index"
		return result
	}

	price, err := index.LookupPrice(*lookup)
	if err != nil {
		log.WithError(err).WithField("address", result.Address).Debug("price lookup failed")
		result.ErrorKind = model.CostErrorNoPrice
		result.ErrorDetail = "no matching price"
		return result
	}

	hourly, monthly := h.CalculateCost(price, index, region, attrs)
	result.HourlyCost = hourly
	result.MonthlyCost = monthly
	result.PriceSource = r.pricing.SourceName(providerID)
	result.Details = describeResource(h, price, attrs)

	return result
}

func describeResource(h handler.ResourceHandler, price *pricing.Price, attrs map[string]any) map[string]string {
	describer, ok := h.(handler.Describer)
	if !ok {
		return nil
	}
	return describer.Describe(price, attrs)
}
