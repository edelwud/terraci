package resourcedef

import (
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

type legacyHandler struct {
	def Definition
}

func (h *legacyHandler) Category() handler.CostCategory {
	return h.def.Category
}

func (h *legacyHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	return h.def.BuildLookup(region, attrs)
}

func (h *legacyHandler) Describe(price *pricing.Price, attrs map[string]any) map[string]string {
	return h.def.DescribeResource(price, attrs)
}

func (h *legacyHandler) CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
	hourly, monthly, _ = h.def.CalculateStandardCost(price, index, region, attrs)
	return hourly, monthly
}

func (h *legacyHandler) CalculateFixedCost(region string, attrs map[string]any) (hourly, monthly float64) {
	hourly, monthly, _ = h.def.CalculateFixedCost(region, attrs)
	return hourly, monthly
}

func (h *legacyHandler) CalculateUsageCost(region string, attrs map[string]any) model.UsageCostEstimate {
	estimate, _ := h.def.CalculateUsageCost(region, attrs)
	return estimate
}

func (h *legacyHandler) SubResources(attrs map[string]any) []handler.SubResource {
	return h.def.BuildSubresources(attrs)
}

// NewLegacyHandler adapts a canonical resource definition to the legacy handler contract.
func NewLegacyHandler(def Definition) (handler.ResourceHandler, error) {
	if err := def.Validate(); err != nil {
		return nil, err
	}
	return &legacyHandler{def: def}, nil
}

// MustLegacyHandler adapts a canonical resource definition and panics on invalid configuration.
func MustLegacyHandler(def Definition) handler.ResourceHandler {
	h, err := NewLegacyHandler(def)
	if err != nil {
		panic(err)
	}
	return h
}

// FromLegacyHandler adapts a legacy handler to the canonical runtime definition.
func FromLegacyHandler(resourceType handler.ResourceType, legacy handler.ResourceHandler) (Definition, error) {
	if legacy == nil {
		return Definition{}, fmt.Errorf("resource definition %q: legacy handler is required", resourceType)
	}

	def := Definition{
		Type:     resourceType,
		Category: legacy.Category(),
	}

	if builder, ok := legacy.(handler.LookupBuilder); ok {
		def.Lookup = builder.BuildLookup
	}
	if describer, ok := legacy.(handler.Describer); ok {
		def.Describe = describer.Describe
	}
	if compound, ok := legacy.(handler.CompoundHandler); ok {
		def.Subresources = compound.SubResources
	}

	switch legacy.Category() {
	case handler.CostCategoryStandard:
		standard, ok := legacy.(handler.StandardCostHandler)
		if !ok {
			return Definition{}, fmt.Errorf("resource definition %q: standard handler does not implement StandardCostHandler", resourceType)
		}
		def.StandardCost = standard.CalculateCost
	case handler.CostCategoryFixed:
		fixed, ok := legacy.(handler.FixedCostHandler)
		if !ok {
			return Definition{}, fmt.Errorf("resource definition %q: fixed handler does not implement FixedCostHandler", resourceType)
		}
		def.FixedCost = fixed.CalculateFixedCost
	case handler.CostCategoryUsageBased:
		usage, ok := legacy.(handler.UsageBasedCostHandler)
		if !ok {
			return Definition{}, fmt.Errorf("resource definition %q: usage handler does not implement UsageBasedCostHandler", resourceType)
		}
		def.UsageCost = usage.CalculateUsageCost
	default:
		return Definition{}, fmt.Errorf("resource definition %q: unsupported category %v", resourceType, legacy.Category())
	}

	if err := def.Validate(); err != nil {
		return Definition{}, err
	}
	return def, nil
}

// MustFromLegacyHandler adapts a legacy handler and panics on invalid configuration.
func MustFromLegacyHandler(resourceType handler.ResourceType, legacy handler.ResourceHandler) Definition {
	def, err := FromLegacyHandler(resourceType, legacy)
	if err != nil {
		panic(err)
	}
	return def
}
