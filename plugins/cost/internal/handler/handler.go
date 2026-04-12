// Package handler provides legacy compatibility aliases and interfaces for
// cloud cost estimation. The canonical value types now live in resourcedef;
// this package re-exports them as type aliases for backward compatibility.
package handler

import (
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

// ResourceType is an alias for resourcedef.ResourceType.
type ResourceType = resourcedef.ResourceType

// CostCategory is an alias for resourcedef.CostCategory.
type CostCategory = resourcedef.CostCategory

// SubResource is an alias for resourcedef.SubResource.
type SubResource = resourcedef.SubResource

const (
	CostCategoryStandard   = resourcedef.CostCategoryStandard
	CostCategoryFixed      = resourcedef.CostCategoryFixed
	CostCategoryUsageBased = resourcedef.CostCategoryUsageBased
)

// ResourceHandler is the shared contract for all pricing handlers.
type ResourceHandler interface {
	Category() CostCategory
}

// StandardCostHandler is implemented by handlers that require a pricing lookup.
type StandardCostHandler interface {
	ResourceHandler
	LookupBuilder
	CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64)
}

// FixedCostHandler is implemented by handlers that can calculate cost without pricing API data.
type FixedCostHandler interface {
	ResourceHandler
	CalculateFixedCost(region string, attrs map[string]any) (hourly, monthly float64)
}

// UsageBasedCostHandler is implemented by handlers whose fixed estimate is usage-derived.
type UsageBasedCostHandler interface {
	ResourceHandler
	CalculateUsageCost(region string, attrs map[string]any) model.UsageCostEstimate
}

// LookupBuilder is implemented by handlers that expose pricing lookup construction.
type LookupBuilder interface {
	BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error)
}

// Describer is implemented by handlers that expose human-readable resource details.
type Describer interface {
	// Describe returns human-readable resource details.
	// price may be nil for Fixed/UsageBased handlers or before API lookup.
	Describe(price *pricing.Price, attrs map[string]any) map[string]string
}

// CompoundHandler is implemented by handlers that produce additional sub-resource costs.
// The estimator dispatches each SubResource to the appropriate handler.
type CompoundHandler interface {
	SubResources(attrs map[string]any) []SubResource
}
