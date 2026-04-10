package runtime

import (
	"context"
	"errors"
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

// ResolveRequest bundles all inputs for a single resource cost resolution.
type ResolveRequest struct {
	ResourceType handler.ResourceType
	Address      string
	Name         string
	ModuleAddr   string
	Region       string
	Attrs        map[string]any
}

// CostResolver handles the cost resolution logic.
type CostResolver struct {
	catalog ProviderCatalogRuntime
	pricing PricingRuntime
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

// NewCostResolver creates a resolver from explicit catalog and pricing dependencies.
func NewCostResolver(catalog ProviderCatalogRuntime, pricingRuntime PricingRuntime) (*CostResolver, error) {
	if catalog == nil {
		return nil, errors.New("cost resolver: provider catalog is required")
	}
	if pricingRuntime == nil {
		return nil, errors.New("cost resolver: pricing runtime is required")
	}
	return &CostResolver{
		catalog: catalog,
		pricing: pricingRuntime,
	}, nil
}

// Resolve calculates cost for a single resource.
func (r *CostResolver) Resolve(ctx context.Context, req ResolveRequest) model.ResourceCost {
	return r.ResolveWithState(ctx, req, nil)
}

// ResolveWithState calculates cost for a single resource using an optional per-resource cache.
func (r *CostResolver) ResolveWithState(ctx context.Context, req ResolveRequest, state *ResolutionState) model.ResourceCost {
	return r.coreResolve(ctx, req, state)
}

func (r *CostResolver) coreResolve(ctx context.Context, req ResolveRequest, state *ResolutionState) model.ResourceCost {
	result := model.ResourceCost{
		Provider:   "",
		Address:    req.Address,
		ModuleAddr: req.ModuleAddr,
		Type:       req.ResourceType.String(),
		Name:       req.Name,
		Region:     req.Region,
		Status:     model.ResourceEstimateStatusExact,
	}

	providerID, ok := r.catalog.ResolveProvider(req.ResourceType)
	if !ok {
		result.Status = model.ResourceEstimateStatusUnsupported
		result.FailureKind = model.FailureKindNoProvider
		result.StatusDetail = "no provider"
		logUnsupportedResource(req.ResourceType.String(), req.Address)
		return result
	}
	result.Provider = providerID

	h, ok := r.catalog.ResolveHandler(providerID, req.ResourceType)
	if !ok {
		result.Status = model.ResourceEstimateStatusUnsupported
		result.FailureKind = model.FailureKindNoHandler
		result.StatusDetail = "no handler"
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
		usageHandler, ok := h.(handler.UsageBasedCostHandler)
		if !ok {
			result.Status = model.ResourceEstimateStatusFailed
			result.FailureKind = model.FailureKindInternal
			result.StatusDetail = "usage-based handler does not implement UsageBasedCostHandler"
			return result
		}
		estimate := usageHandler.CalculateUsageCost(req.Region, attrs)
		result.HourlyCost = estimate.HourlyCost
		result.MonthlyCost = estimate.MonthlyCost
		result.Status = usageEstimateStatus(estimate)
		result.StatusDetail = usageEstimateDetail(estimate)
		result.PriceSource = priceSourceUsageBased
		return result
	case handler.CostCategoryFixed:
		fixedHandler, ok := h.(handler.FixedCostHandler)
		if !ok {
			result.Status = model.ResourceEstimateStatusFailed
			result.FailureKind = model.FailureKindInternal
			result.StatusDetail = "fixed-cost handler does not implement FixedCostHandler"
			return result
		}
		hourly, monthly := fixedHandler.CalculateFixedCost(req.Region, attrs)
		result.HourlyCost = hourly
		result.MonthlyCost = monthly
		result.Status = model.ResourceEstimateStatusExact
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
		result.Status = model.ResourceEstimateStatusFailed
		result.FailureKind = model.FailureKindInternal
		result.StatusDetail = fmt.Sprintf("unknown cost category: %d", h.Category())
		return result
	}
}

// ResolveBeforeCost calculates the before-state cost for update/replace resources.
func (r *CostResolver) ResolveBeforeCost(ctx context.Context, rc *model.ResourceCost, resourceType handler.ResourceType, beforeAttrs map[string]any, region string) {
	r.ResolveBeforeCostWithState(ctx, rc, resourceType, beforeAttrs, region, nil)
}

// ResolveBeforeCostWithState calculates the before-state cost using an optional per-resource cache.
func (r *CostResolver) ResolveBeforeCostWithState(ctx context.Context, rc *model.ResourceCost, resourceType handler.ResourceType, beforeAttrs map[string]any, region string, state *ResolutionState) {
	providerID, ok := r.catalog.ResolveProvider(resourceType)
	if !ok {
		return
	}
	h, ok := r.catalog.ResolveHandler(providerID, resourceType)
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
		fixedHandler, ok := h.(handler.FixedCostHandler)
		if !ok {
			rc.Status = model.ResourceEstimateStatusFailed
			rc.FailureKind = model.FailureKindInternal
			rc.StatusDetail = "fixed-cost handler does not implement FixedCostHandler"
			return
		}
		hourly, monthly := fixedHandler.CalculateFixedCost(region, beforeAttrs)
		rc.BeforeHourlyCost = hourly
		rc.BeforeMonthlyCost = monthly
	case handler.CostCategoryUsageBased:
		// Usage-based resources (e.g. data transfer, Lambda invocations) require
		// runtime telemetry that is unavailable at plan time; skip silently.
	default:
		rc.Status = model.ResourceEstimateStatusFailed
		rc.FailureKind = model.FailureKindInternal
		rc.StatusDetail = fmt.Sprintf("unknown cost category: %d", h.Category())
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

	providerID, ok := r.catalog.ResolveProvider(req.ResourceType)
	if !ok {
		return results
	}

	h, ok := r.catalog.ResolveHandler(providerID, req.ResourceType)
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

	standardHandler, ok := sc.handler.(handler.StandardCostHandler)
	if !ok {
		result.Status = model.ResourceEstimateStatusFailed
		result.FailureKind = model.FailureKindLookupFailed
		result.StatusDetail = "standard handler does not implement StandardCostHandler"
		return result
	}

	lookup, err := standardHandler.BuildLookup(sc.region, sc.attrs)
	if err != nil {
		result.Status = model.ResourceEstimateStatusFailed
		result.FailureKind = model.FailureKindLookupFailed
		result.StatusDetail = err.Error()
		return result
	}
	if lookup == nil {
		result.Status = model.ResourceEstimateStatusExact
		return result
	}

	index, err := r.getIndex(ctx, lookup.ServiceID, sc.region, sc.state)
	if err != nil {
		log.WithError(err).WithField("service", lookup.ServiceID.String()).WithField("region", sc.region).Debug("failed to get pricing index")
		result.Status = model.ResourceEstimateStatusFailed
		result.FailureKind = model.FailureKindAPIFailure
		result.StatusDetail = "pricing unavailable"
		return result
	}
	if index == nil {
		result.Status = model.ResourceEstimateStatusFailed
		result.FailureKind = model.FailureKindAPIFailure
		result.StatusDetail = "empty pricing index"
		return result
	}

	price, err := index.LookupPrice(*lookup)
	if err != nil {
		log.WithError(err).WithField("address", result.Address).Debug("price lookup failed")
		result.Status = model.ResourceEstimateStatusFailed
		result.FailureKind = model.FailureKindNoPrice
		result.StatusDetail = "no matching price"
		return result
	}

	hourly, monthly := standardHandler.CalculateCost(price, index, sc.region, sc.attrs)
	result.HourlyCost = hourly
	result.MonthlyCost = monthly
	result.Status = model.ResourceEstimateStatusExact
	result.PriceSource = r.pricing.SourceName(sc.providerID)
	result.Details = describeResource(sc.handler, price, sc.attrs)

	return result
}

func (r *CostResolver) getIndex(ctx context.Context, service pricing.ServiceID, region string, state *ResolutionState) (*pricing.PriceIndex, error) {
	if state != nil {
		if idx, ok := state.indexes[indexCacheKey(service, region)]; ok {
			return idx, nil
		}
	}

	idx, err := r.pricing.GetIndex(ctx, service, region)
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

func usageEstimateDetail(estimate model.UsageCostEstimate) string {
	if estimate.Detail != "" {
		return estimate.Detail
	}

	switch usageEstimateStatus(estimate) {
	case model.ResourceEstimateStatusUsageEstimated:
		return "usage-based estimate derived from configured capacity"
	case model.ResourceEstimateStatusUsageUnknown, model.ResourceEstimateStatusExact,
		model.ResourceEstimateStatusUnsupported, model.ResourceEstimateStatusFailed:
		return priceSourceUsageBased
	default:
		return priceSourceUsageBased
	}
}

func usageEstimateStatus(estimate model.UsageCostEstimate) model.ResourceEstimateStatus {
	switch estimate.Status {
	case model.ResourceEstimateStatusUsageEstimated, model.ResourceEstimateStatusUsageUnknown:
		return estimate.Status
	case model.ResourceEstimateStatusExact, model.ResourceEstimateStatusUnsupported, model.ResourceEstimateStatusFailed:
		if !model.CostIsZero(estimate.MonthlyCost) || !model.CostIsZero(estimate.HourlyCost) {
			return model.ResourceEstimateStatusUsageEstimated
		}
		return model.ResourceEstimateStatusUsageUnknown
	default:
		if !model.CostIsZero(estimate.MonthlyCost) || !model.CostIsZero(estimate.HourlyCost) {
			return model.ResourceEstimateStatusUsageEstimated
		}
		return model.ResourceEstimateStatusUsageUnknown
	}
}
