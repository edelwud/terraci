// Package provider defines provider-agnostic interfaces for cloud cost estimation.
// AWS, GCP, Azure handlers all implement these interfaces.
package provider

import "github.com/edelwud/terraci/plugins/cost/internal/pricing"

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

// ResourceHandler extracts pricing information from terraform resource attributes.
// Implemented by each cloud provider's resource handlers.
type ResourceHandler interface {
	// Category returns how this handler calculates costs.
	Category() CostCategory
	// ServiceCode returns the cloud service code for pricing API.
	ServiceCode() pricing.ServiceCode
	// BuildLookup creates a PriceLookup from terraform resource attributes.
	// Not called for Fixed or UsageBased handlers.
	BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error)
	// CalculateCost calculates monthly cost from price, index, and resource attributes.
	// For Fixed/UsageBased handlers, price and index may be nil.
	CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64)
	// Describe returns human-readable resource details.
	// price may be nil for Fixed/UsageBased handlers or before API lookup.
	Describe(price *pricing.Price, attrs map[string]any) map[string]string
}

// SubResource represents a virtual sub-resource synthesized from a parent resource's
// inline attributes (e.g., root_block_device inside aws_instance → aws_ebs_volume).
type SubResource struct {
	Suffix string         // Address suffix, e.g., "/root_volume"
	Type   string         // Resource type for handler lookup, e.g., "aws_ebs_volume"
	Attrs  map[string]any // Translated attributes for the sub-resource handler
}

// CompoundHandler is implemented by handlers that produce additional sub-resource costs.
// The estimator dispatches each SubResource to the appropriate handler.
type CompoundHandler interface {
	SubResources(attrs map[string]any) []SubResource
}
