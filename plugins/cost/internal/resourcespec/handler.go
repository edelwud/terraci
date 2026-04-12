package resourcespec

import (
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

type baseHandler struct {
	spec ResourceSpec
}

func (h *baseHandler) Category() handler.CostCategory {
	return h.spec.Category
}

func (h *baseHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	if h.spec.Lookup == nil || h.spec.Lookup.BuildFunc == nil {
		return nil, nil
	}
	return h.spec.Lookup.BuildFunc(region, attrs)
}

func (h *baseHandler) Describe(price *pricing.Price, attrs map[string]any) map[string]string {
	if h.spec.Describe == nil {
		return nil
	}
	return h.spec.Describe.Describe(price, attrs)
}

func (h *baseHandler) SubResources(attrs map[string]any) []handler.SubResource {
	if h.spec.Subresources == nil || h.spec.Subresources.BuildFunc == nil {
		return nil
	}
	return h.spec.Subresources.BuildFunc(attrs)
}

type standardHandler struct {
	*baseHandler
}

func (h *standardHandler) CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
	return h.spec.Standard.CostFunc(price, index, region, attrs)
}

type fixedHandler struct {
	*baseHandler
}

func (h *fixedHandler) CalculateFixedCost(region string, attrs map[string]any) (hourly, monthly float64) {
	return h.spec.Fixed.CostFunc(region, attrs)
}

type usageHandler struct {
	*baseHandler
}

func (h *usageHandler) CalculateUsageCost(region string, attrs map[string]any) model.UsageCostEstimate {
	return h.spec.Usage.EstimateFunc(region, attrs)
}

// NewHandler compiles a typed resource spec into a runtime handler.
func NewHandler(spec ResourceSpec) (handler.ResourceHandler, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	base := &baseHandler{spec: spec}
	switch spec.Category {
	case handler.CostCategoryStandard:
		return &standardHandler{baseHandler: base}, nil
	case handler.CostCategoryFixed:
		return &fixedHandler{baseHandler: base}, nil
	case handler.CostCategoryUsageBased:
		return &usageHandler{baseHandler: base}, nil
	default:
		return nil, spec.Validate()
	}
}

// MustHandler compiles a resource spec and panics on invalid configuration.
func MustHandler(spec ResourceSpec) handler.ResourceHandler {
	h, err := NewHandler(spec)
	if err != nil {
		panic(err)
	}
	return h
}
