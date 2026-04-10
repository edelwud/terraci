// Package handler defines provider-agnostic interfaces for cloud cost estimation.
// AWS, GCP, Azure handlers all implement these interfaces.
package handler

import (
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ResourceType is a provider-neutral Terraform resource identifier.
type ResourceType string

// String returns the raw Terraform resource type value.
func (r ResourceType) String() string { return string(r) }

// RuntimeDeps stores an optional provider runtime dependency for a handler.
// Zero value is valid and can be paired with a fallback in tests.
type RuntimeDeps[T any] struct {
	Runtime *T
}

// RuntimeOr returns the injected runtime or the provided fallback when unset.
func (d RuntimeDeps[T]) RuntimeOr(fallback *T) *T {
	if d.Runtime != nil {
		return d.Runtime
	}
	return fallback
}

// CostCategory classifies how a handler calculates costs.
type CostCategory int

const (
	// CostCategoryStandard requires pricing API lookup.
	CostCategoryStandard CostCategory = iota
	// CostCategoryFixed uses hardcoded costs (no API call needed).
	CostCategoryFixed
	// CostCategoryUsageBased is usage-based pricing (returns $0 for fixed estimates).
	CostCategoryUsageBased
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

// SubResource represents a virtual sub-resource synthesized from a parent resource's
// inline attributes (e.g., root_block_device inside aws_instance → aws_ebs_volume).
type SubResource struct {
	Suffix string         // Address suffix, e.g., "/root_volume"
	Type   ResourceType   // Resource type for handler lookup, e.g., "aws_ebs_volume"
	Attrs  map[string]any // Translated attributes for the sub-resource handler
}

// CompoundHandler is implemented by handlers that produce additional sub-resource costs.
// The estimator dispatches each SubResource to the appropriate handler.
type CompoundHandler interface {
	SubResources(attrs map[string]any) []SubResource
}
