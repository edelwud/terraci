package resourcedef

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// LookupFunc builds a pricing lookup for one resource instance.
type LookupFunc func(region string, attrs map[string]any) (*pricing.PriceLookup, error)

// DescribeFunc builds human-readable resource detail fields.
type DescribeFunc func(price *pricing.Price, attrs map[string]any) map[string]string

// StandardCostFunc calculates cost from resolved pricing data.
type StandardCostFunc func(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64)

// FixedCostFunc calculates cost without external pricing data.
type FixedCostFunc func(region string, attrs map[string]any) (hourly, monthly float64)

// UsageCostFunc calculates a usage-based estimate.
type UsageCostFunc func(region string, attrs map[string]any) model.UsageCostEstimate

// SubresourceFunc synthesizes subresources from resource attributes.
type SubresourceFunc func(attrs map[string]any) []handler.SubResource

// Definition is the canonical runtime execution contract for one resource type.
type Definition struct {
	Type         handler.ResourceType
	Category     handler.CostCategory
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

	switch d.Category {
	case handler.CostCategoryStandard:
		if d.StandardCost == nil {
			return fmt.Errorf("resource definition %q: standard cost function is required", d.Type)
		}
	case handler.CostCategoryFixed:
		if d.FixedCost == nil {
			return fmt.Errorf("resource definition %q: fixed cost function is required", d.Type)
		}
	case handler.CostCategoryUsageBased:
		if d.UsageCost == nil {
			return fmt.Errorf("resource definition %q: usage cost function is required", d.Type)
		}
	default:
		return fmt.Errorf("resource definition %q: unsupported category %v", d.Type, d.Category)
	}

	return nil
}

// BuildLookup builds a pricing lookup when the resource needs one.
func (d Definition) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	if d.Lookup == nil {
		return nil, nil
	}
	return d.Lookup(region, attrs)
}

// DescribeResource returns human-readable detail fields for a resource.
func (d Definition) DescribeResource(price *pricing.Price, attrs map[string]any) map[string]string {
	if d.Describe == nil {
		return nil
	}
	return d.Describe(price, attrs)
}

// CalculateStandardCost evaluates standard pricing when available.
func (d Definition) CalculateStandardCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64, ok bool) {
	if d.StandardCost == nil {
		return 0, 0, false
	}
	hourly, monthly = d.StandardCost(price, index, region, attrs)
	return hourly, monthly, true
}

// CalculateFixedCost evaluates fixed pricing when available.
func (d Definition) CalculateFixedCost(region string, attrs map[string]any) (hourly, monthly float64, ok bool) {
	if d.FixedCost == nil {
		return 0, 0, false
	}
	hourly, monthly = d.FixedCost(region, attrs)
	return hourly, monthly, true
}

// CalculateUsageCost evaluates usage-based pricing when available.
func (d Definition) CalculateUsageCost(region string, attrs map[string]any) (model.UsageCostEstimate, bool) {
	if d.UsageCost == nil {
		return model.UsageCostEstimate{}, false
	}
	return d.UsageCost(region, attrs), true
}

// BuildSubresources returns synthesized subresources when configured.
func (d Definition) BuildSubresources(attrs map[string]any) []handler.SubResource {
	if d.Subresources == nil {
		return nil
	}
	return d.Subresources(attrs)
}
