package resourcedef

import (
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// legacyCategorizer is the minimal interface for legacy handler category detection.
type legacyCategorizer interface {
	Category() CostCategory
}

// legacyLookupBuilder matches the legacy LookupBuilder interface.
type legacyLookupBuilder interface {
	BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error)
}

// legacyDescriber matches the legacy Describer interface.
type legacyDescriber interface {
	Describe(price *pricing.Price, attrs map[string]any) map[string]string
}

// legacyStandardCostHandler matches the legacy StandardCostHandler interface.
type legacyStandardCostHandler interface {
	CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64)
}

// legacyFixedCostHandler matches the legacy FixedCostHandler interface.
type legacyFixedCostHandler interface {
	CalculateFixedCost(region string, attrs map[string]any) (hourly, monthly float64)
}

// legacyUsageBasedCostHandler matches the legacy UsageBasedCostHandler interface.
type legacyUsageBasedCostHandler interface {
	CalculateUsageCost(region string, attrs map[string]any) model.UsageCostEstimate
}

// legacyCompoundHandler matches the legacy CompoundHandler interface.
type legacyCompoundHandler interface {
	SubResources(attrs map[string]any) []SubResource
}

type legacyHandler struct {
	def Definition
}

func (h *legacyHandler) Category() CostCategory {
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

func (h *legacyHandler) SubResources(attrs map[string]any) []SubResource {
	return h.def.BuildSubresources(attrs)
}

// LegacyHandler is the exported type for adapted definitions.
// It implements all legacy handler capability interfaces via duck typing.
type LegacyHandler = legacyHandler

// NewLegacyHandler adapts a canonical resource definition to the legacy handler contract.
// The returned *LegacyHandler implements Category(), BuildLookup(), Describe(),
// CalculateCost(), CalculateFixedCost(), CalculateUsageCost(), and SubResources().
func NewLegacyHandler(def Definition) (*LegacyHandler, error) {
	if err := def.Validate(); err != nil {
		return nil, err
	}
	return &legacyHandler{def: def}, nil
}

// MustLegacyHandler adapts a canonical resource definition and panics on invalid configuration.
func MustLegacyHandler(def Definition) *LegacyHandler {
	h, err := NewLegacyHandler(def)
	if err != nil {
		panic(err)
	}
	return h
}

// FromLegacyHandler adapts a legacy handler to the canonical runtime definition.
// The handler must implement legacyCategorizer (Category() CostCategory).
func FromLegacyHandler(resourceType ResourceType, legacy legacyCategorizer) (Definition, error) {
	if legacy == nil {
		return Definition{}, fmt.Errorf("resource definition %q: legacy handler is required", resourceType)
	}

	def := Definition{
		Type:     resourceType,
		Category: legacy.Category(),
	}

	if builder, ok := legacy.(legacyLookupBuilder); ok {
		def.Lookup = builder.BuildLookup
	}
	if describer, ok := legacy.(legacyDescriber); ok {
		def.Describe = describer.Describe
	}
	if compound, ok := legacy.(legacyCompoundHandler); ok {
		def.Subresources = compound.SubResources
	}

	switch legacy.Category() {
	case CostCategoryStandard:
		standard, ok := legacy.(legacyStandardCostHandler)
		if !ok {
			return Definition{}, fmt.Errorf("resource definition %q: standard handler does not implement StandardCostHandler", resourceType)
		}
		def.StandardCost = standard.CalculateCost
	case CostCategoryFixed:
		fixed, ok := legacy.(legacyFixedCostHandler)
		if !ok {
			return Definition{}, fmt.Errorf("resource definition %q: fixed handler does not implement FixedCostHandler", resourceType)
		}
		def.FixedCost = fixed.CalculateFixedCost
	case CostCategoryUsageBased:
		usage, ok := legacy.(legacyUsageBasedCostHandler)
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
func MustFromLegacyHandler(resourceType ResourceType, legacy legacyCategorizer) Definition {
	def, err := FromLegacyHandler(resourceType, legacy)
	if err != nil {
		panic(err)
	}
	return def
}
