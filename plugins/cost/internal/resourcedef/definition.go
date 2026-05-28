// Package resourcedef defines the canonical runtime contract for resource cost estimation.
// It owns the core value types (ResourceType, CostCategory, SubResource) and the
// Definition struct used by provider resource implementations.
package resourcedef

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ResourceType is a provider-neutral Terraform resource identifier.
type ResourceType string

// String returns the raw Terraform resource type value.
func (r ResourceType) String() string { return string(r) }

// CostCategory classifies how a resource definition calculates costs.
type CostCategory int

const (
	// CostCategoryStandard requires pricing API lookup.
	CostCategoryStandard CostCategory = iota
	// CostCategoryFixed uses hardcoded costs (no API call needed).
	CostCategoryFixed
	// CostCategoryUsageBased is usage-based pricing (returns $0 for fixed estimates).
	CostCategoryUsageBased
)

// SubResource represents a virtual sub-resource synthesized from a parent resource's
// inline attributes (e.g., root_block_device inside aws_instance → aws_ebs_volume).
type SubResource struct {
	Suffix string       // Address suffix, e.g., "/root_volume"
	Type   ResourceType // Resource type for definition lookup, e.g., "aws_ebs_volume"
	Attrs  RawAttrs
}

// ParseFunc parses raw Terraform attributes into a resource-specific typed payload.
type ParseFunc func(attrs RawAttrs) (Attributes, error)

// LookupFunc builds a pricing lookup for one resource instance.
type LookupFunc func(region string, attrs Attributes) (*pricing.PriceLookup, error)

// DescribeFunc builds human-readable resource detail fields.
type DescribeFunc func(price *pricing.Price, attrs Attributes) map[string]string

// StandardCostFunc calculates cost from resolved pricing data.
type StandardCostFunc func(price *pricing.Price, index *pricing.PriceIndex, region string, attrs Attributes) (hourly, monthly float64)

// FixedCostFunc calculates cost without external pricing data.
type FixedCostFunc func(region string, attrs Attributes) (hourly, monthly float64)

// UsageCostFunc calculates a usage-based estimate.
type UsageCostFunc func(region string, attrs Attributes) model.UsageCostEstimate

// SubresourceFunc synthesizes subresources from resource attributes.
type SubresourceFunc func(attrs Attributes) []SubResource

// Definition is the canonical runtime execution contract for one resource type.
type Definition struct {
	Type         ResourceType
	Category     CostCategory
	Parse        ParseFunc
	Lookup       LookupFunc
	Describe     DescribeFunc
	StandardCost StandardCostFunc
	FixedCost    FixedCostFunc
	UsageCost    UsageCostFunc
	Subresources SubresourceFunc
}

// Validate ensures the definition is internally consistent for its category.
func (d Definition) Validate() error {
	if d.Type == "" {
		return errors.New("resource definition: type is required")
	}
	if d.Parse == nil {
		return fmt.Errorf("resource definition %q: parse function is required", d.Type)
	}

	switch d.Category {
	case CostCategoryStandard:
		if d.StandardCost == nil {
			return fmt.Errorf("resource definition %q: standard cost function is required", d.Type)
		}
	case CostCategoryFixed:
		if d.FixedCost == nil {
			return fmt.Errorf("resource definition %q: fixed cost function is required", d.Type)
		}
	case CostCategoryUsageBased:
		if d.UsageCost == nil {
			return fmt.Errorf("resource definition %q: usage cost function is required", d.Type)
		}
	default:
		return fmt.Errorf("resource definition %q: unsupported category %v", d.Type, d.Category)
	}

	return nil
}

// ParseAttrs parses raw Terraform attributes into the definition's typed payload.
func (d Definition) ParseAttrs(attrs RawAttrs) (Attributes, error) {
	if d.Parse == nil {
		return Attributes{}, fmt.Errorf("resource definition %q: parse function is required", d.Type)
	}
	parsed, err := d.Parse(attrs)
	if err != nil {
		return Attributes{}, fmt.Errorf("parse attributes for %q: %w", d.Type, err)
	}
	return parsed, nil
}

// BuildLookup builds a pricing lookup when the resource needs one.
func (d Definition) BuildLookup(region string, attrs Attributes) (*pricing.PriceLookup, error) {
	if d.Lookup == nil {
		return nil, nil
	}
	return d.Lookup(region, attrs)
}

// DescribeResource returns human-readable detail fields for a resource.
func (d Definition) DescribeResource(price *pricing.Price, attrs Attributes) map[string]string {
	if d.Describe == nil {
		return nil
	}
	return d.Describe(price, attrs)
}

// CalculateStandardCost evaluates standard pricing when available.
func (d Definition) CalculateStandardCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs Attributes) (hourly, monthly float64, ok bool) {
	if d.StandardCost == nil {
		return 0, 0, false
	}
	hourly, monthly = d.StandardCost(price, index, region, attrs)
	return hourly, monthly, true
}

// CalculateFixedCost evaluates fixed pricing when available.
func (d Definition) CalculateFixedCost(region string, attrs Attributes) (hourly, monthly float64, ok bool) {
	if d.FixedCost == nil {
		return 0, 0, false
	}
	hourly, monthly = d.FixedCost(region, attrs)
	return hourly, monthly, true
}

// CalculateUsageCost evaluates usage-based pricing when available.
func (d Definition) CalculateUsageCost(region string, attrs Attributes) (model.UsageCostEstimate, bool) {
	if d.UsageCost == nil {
		return model.UsageCostEstimate{}, false
	}
	return d.UsageCost(region, attrs), true
}

// BuildSubresources returns synthesized subresources when configured.
func (d Definition) BuildSubresources(attrs Attributes) []SubResource {
	if d.Subresources == nil {
		return nil
	}
	return d.Subresources(attrs)
}
