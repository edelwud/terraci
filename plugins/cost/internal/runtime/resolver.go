package runtime

import (
	"context"
	"fmt"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const priceSourceUsageBased = "usage-based"

// ProviderCatalogRuntime exposes provider ownership and handler lookup for resolution and planning.
type ProviderCatalogRuntime interface {
	ResolveProvider(resourceType handler.ResourceType) (string, bool)
	ResolveHandler(providerID string, resourceType handler.ResourceType) (handler.ResourceHandler, bool)
}

// PricingRuntime exposes provider-scoped pricing access for resolution.
type PricingRuntime interface {
	GetIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error)
	SourceName(providerID string) string
}

// ResolutionRuntime exposes the provider-aware runtime surface required by cost resolution.
type ResolutionRuntime interface {
	ProviderCatalogRuntime
	PricingRuntime
}

type resolutionRuntime struct {
	catalog ProviderCatalogRuntime
	pricing PricingRuntime
}

// NewResolutionRuntime combines provider catalog and pricing runtime into one resolver runtime.
func NewResolutionRuntime(catalog ProviderCatalogRuntime, pricingRuntime PricingRuntime) ResolutionRuntime {
	return resolutionRuntime{catalog: catalog, pricing: pricingRuntime}
}

func (r resolutionRuntime) ResolveProvider(resourceType handler.ResourceType) (string, bool) {
	return r.catalog.ResolveProvider(resourceType)
}

func (r resolutionRuntime) ResolveHandler(providerID string, resourceType handler.ResourceType) (handler.ResourceHandler, bool) {
	return r.catalog.ResolveHandler(providerID, resourceType)
}

func (r resolutionRuntime) GetIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
	return r.pricing.GetIndex(ctx, service, region)
}

func (r resolutionRuntime) SourceName(providerID string) string {
	return r.pricing.SourceName(providerID)
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
	runtime    ResolutionRuntime
	middleware []CostMiddleware
}

// ResolutionState stores per-resource pricing lookups that can be reused across
// current/before-state resolution without changing handler contracts.
type ResolutionState struct {
	indexes map[string]*pricing.PriceIndex
}

// NewResolutionState creates an empty per-resource resolution cache.
func NewResolutionState() *ResolutionState {
	return &ResolutionState{indexes: make(map[string]*pricing.PriceIndex)}
}

// NewCostResolver creates a new resolver with the given provider-aware runtime.
func NewCostResolver(runtime ResolutionRuntime) *CostResolver {
	return &CostResolver{
		runtime: runtime,
	}
}

// Use appends a middleware to the resolver chain.
func (r *CostResolver) Use(mw CostMiddleware) {
	r.middleware = append(r.middleware, mw)
}

// Resolve calculates cost for a single resource.
func (r *CostResolver) Resolve(ctx context.Context, req ResolveRequest) model.ResourceCost {
	return r.ResolveWithState(ctx, req, nil)
}

// ResolveWithState calculates cost for a single resource using an optional per-resource cache.
func (r *CostResolver) ResolveWithState(ctx context.Context, req ResolveRequest, state *ResolutionState) model.ResourceCost {
	if len(r.middleware) > 0 {
		return r.resolveWithMiddleware(ctx, req, state)
	}
	return r.coreResolve(ctx, req, state)
}

func (r *CostResolver) resolveWithMiddleware(ctx context.Context, req ResolveRequest, state *ResolutionState) model.ResourceCost {
	fn := func(ctx context.Context, req ResolveRequest) model.ResourceCost {
		return r.coreResolve(ctx, req, state)
	}
	for i := len(r.middleware) - 1; i >= 0; i-- {
		mw := r.middleware[i]
		next := fn
		fn = func(ctx context.Context, req ResolveRequest) model.ResourceCost {
			return mw(ctx, next, req)
		}
	}
	return fn(ctx, req)
}

func (r *CostResolver) coreResolve(ctx context.Context, req ResolveRequest, state *ResolutionState) model.ResourceCost {
	result := model.ResourceCost{
		Provider:   "",
		Address:    req.Address,
		ModuleAddr: req.ModuleAddr,
		Type:       req.ResourceType.String(),
		Name:       req.Name,
		Region:     req.Region,
	}

	providerID, ok := r.runtime.ResolveProvider(req.ResourceType)
	if !ok {
		result.ErrorKind = model.CostErrorNoProvider
		result.ErrorDetail = "no provider"
		logUnsupportedResource(req.ResourceType.String(), req.Address)
		return result
	}
	result.Provider = providerID

	h, ok := r.runtime.ResolveHandler(providerID, req.ResourceType)
	if !ok {
		result.ErrorKind = model.CostErrorNoHandler
		result.ErrorDetail = "no handler"
		logUnsupportedResource(req.ResourceType.String(), req.Address)
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
		return r.resolveStandardCost(ctx, standardResolutionCtx{
			providerID: providerID,
			handler:    h,
			attrs:      attrs,
			region:     req.Region,
			result:     result,
			state:      state,
		})
	default:
		result.ErrorKind = model.CostErrorInternal
		result.ErrorDetail = fmt.Sprintf("unknown cost category: %d", h.Category())
		return result
	}
}

// ResolveBeforeCost calculates the before-state cost for update/replace resources.
func (r *CostResolver) ResolveBeforeCost(ctx context.Context, rc *model.ResourceCost, resourceType handler.ResourceType, beforeAttrs map[string]any, region string) {
	r.ResolveBeforeCostWithState(ctx, rc, resourceType, beforeAttrs, region, nil)
}

// ResolveBeforeCostWithState calculates the before-state cost using an optional per-resource cache.
func (r *CostResolver) ResolveBeforeCostWithState(ctx context.Context, rc *model.ResourceCost, resourceType handler.ResourceType, beforeAttrs map[string]any, region string, state *ResolutionState) {
	providerID, ok := r.runtime.ResolveProvider(resourceType)
	if !ok {
		return
	}
	h, ok := r.runtime.ResolveHandler(providerID, resourceType)
	if !ok {
		return
	}

	switch h.Category() {
	case handler.CostCategoryStandard:
		before := r.resolveStandardCost(ctx, standardResolutionCtx{
			providerID: providerID,
			handler:    h,
			attrs:      beforeAttrs,
			region:     region,
			result:     model.ResourceCost{Provider: providerID},
			state:      state,
		})
		rc.BeforeHourlyCost = before.HourlyCost
		rc.BeforeMonthlyCost = before.MonthlyCost
	case handler.CostCategoryFixed:
		hourly, monthly := h.CalculateCost(nil, nil, region, beforeAttrs)
		rc.BeforeHourlyCost = hourly
		rc.BeforeMonthlyCost = monthly
	case handler.CostCategoryUsageBased:
		// Usage-based resources (e.g. data transfer, Lambda invocations) require
		// runtime telemetry that is unavailable at plan time; skip silently.
	default:
		rc.ErrorKind = model.CostErrorInternal
		rc.ErrorDetail = fmt.Sprintf("unknown cost category: %d", h.Category())
	}
}

// ResolveWithSubResources resolves a resource and any compound sub-resources.
func (r *CostResolver) ResolveWithSubResources(ctx context.Context, req ResolveRequest) []model.ResourceCost {
	return r.ResolveWithSubResourcesState(ctx, req, nil)
}

// ResolveWithSubResourcesState resolves a resource and any sub-resources using an optional per-resource cache.
func (r *CostResolver) ResolveWithSubResourcesState(ctx context.Context, req ResolveRequest, state *ResolutionState) []model.ResourceCost {
	primary := r.ResolveWithState(ctx, req, state)
	results := []model.ResourceCost{primary}

	providerID, ok := r.runtime.ResolveProvider(req.ResourceType)
	if !ok {
		return results
	}

	h, ok := r.runtime.ResolveHandler(providerID, req.ResourceType)
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
		results = append(results, r.ResolveWithState(ctx, subReq, state))
	}

	return results
}

// standardResolutionCtx carries the per-call context for resolveStandardCost,
// reducing the parameter count and making call sites readable.
type standardResolutionCtx struct {
	providerID string
	handler    handler.ResourceHandler
	attrs      map[string]any
	region     string
	result     model.ResourceCost
	state      *ResolutionState
}

func (r *CostResolver) resolveStandardCost(ctx context.Context, sc standardResolutionCtx) model.ResourceCost {
	result := sc.result

	lookupBuilder, ok := sc.handler.(handler.LookupBuilder)
	if !ok {
		result.ErrorKind = model.CostErrorLookupFailed
		result.ErrorDetail = "lookup builder not implemented"
		return result
	}

	lookup, err := lookupBuilder.BuildLookup(sc.region, sc.attrs)
	if err != nil {
		result.ErrorKind = model.CostErrorLookupFailed
		result.ErrorDetail = err.Error()
		return result
	}
	if lookup == nil {
		return result
	}

	index, err := r.getIndex(ctx, lookup.ServiceID, sc.region, sc.state)
	if err != nil {
		log.WithError(err).WithField("service", lookup.ServiceID.String()).WithField("region", sc.region).Debug("failed to get pricing index")
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

	hourly, monthly := sc.handler.CalculateCost(price, index, sc.region, sc.attrs)
	result.HourlyCost = hourly
	result.MonthlyCost = monthly
	result.PriceSource = r.runtime.SourceName(sc.providerID)
	result.Details = describeResource(sc.handler, price, sc.attrs)

	return result
}

func (r *CostResolver) getIndex(ctx context.Context, service pricing.ServiceID, region string, state *ResolutionState) (*pricing.PriceIndex, error) {
	if state != nil {
		if idx, ok := state.indexes[indexCacheKey(service, region)]; ok {
			return idx, nil
		}
	}

	idx, err := r.runtime.GetIndex(ctx, service, region)
	if err != nil {
		return nil, err
	}
	if state != nil && idx != nil {
		state.indexes[indexCacheKey(service, region)] = idx
	}
	return idx, nil
}

func indexCacheKey(service pricing.ServiceID, region string) string {
	return service.String() + "|" + region
}

func describeResource(h handler.ResourceHandler, price *pricing.Price, attrs map[string]any) map[string]string {
	describer, ok := h.(handler.Describer)
	if !ok {
		return nil
	}
	return describer.Describe(price, attrs)
}

// logUnsupportedResource emits a debug-level trace when a resource type has no registered handler.
func logUnsupportedResource(resourceType, address string) {
	log.WithField("type", resourceType).
		WithField("address", address).
		Debug("resource type not supported for cost estimation")
}
