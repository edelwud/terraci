package resourcespec

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

// ValueFunc resolves a string field from pricing context and resource attributes.
type ValueFunc func(price *pricing.Price, attrs map[string]any) (string, bool)

// LookupBuildFunc builds a pricing lookup for one resource instance.
type LookupBuildFunc func(region string, attrs map[string]any) (*pricing.PriceLookup, error)

// DescribeBuildFunc builds resource detail fields for presentation.
type DescribeBuildFunc func(price *pricing.Price, attrs map[string]any) map[string]string

// StandardCostFunc calculates cost from resolved pricing data.
type StandardCostFunc func(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64)

// FixedCostFunc calculates cost without external pricing data.
type FixedCostFunc func(region string, attrs map[string]any) (hourly, monthly float64)

// UsageEstimateFunc calculates a usage-based estimate.
type UsageEstimateFunc func(region string, attrs map[string]any) model.UsageCostEstimate

// SubresourceBuildFunc synthesizes subresources from resource attributes.
type SubresourceBuildFunc func(attrs map[string]any) []handler.SubResource

// LookupSpec declares pricing lookup behavior.
type LookupSpec struct {
	BuildFunc LookupBuildFunc
}

// DescribeField declares one detail field in a resource description.
type DescribeField struct {
	Key       string
	Value     ValueFunc
	OmitEmpty bool
}

// DescribeSpec declares resource description behavior.
type DescribeSpec struct {
	Fields    []DescribeField
	BuildFunc DescribeBuildFunc
}

// StandardPricingSpec declares standard price-index-based pricing behavior.
type StandardPricingSpec struct {
	CostFunc StandardCostFunc
}

// FixedPricingSpec declares fixed pricing behavior.
type FixedPricingSpec struct {
	CostFunc FixedCostFunc
}

// UsagePricingSpec declares usage-based pricing behavior.
type UsagePricingSpec struct {
	EstimateFunc UsageEstimateFunc
}

// SubresourceSpec declares compound-resource expansion behavior.
type SubresourceSpec struct {
	BuildFunc SubresourceBuildFunc
}

// ResourceSpec is a provider-agnostic declaration of one resource's pricing behavior.
type ResourceSpec struct {
	Type         handler.ResourceType
	Category     handler.CostCategory
	Lookup       *LookupSpec
	Describe     *DescribeSpec
	Standard     *StandardPricingSpec
	Fixed        *FixedPricingSpec
	Usage        *UsagePricingSpec
	Subresources *SubresourceSpec
}

// Validate ensures the spec is internally consistent.
func (s ResourceSpec) Validate() error {
	if s.Type == "" {
		return errors.New("resource spec: type is required")
	}

	if err := s.validateCategory(); err != nil {
		return err
	}
	if err := s.validateLookup(); err != nil {
		return err
	}
	if err := s.validateDescribe(); err != nil {
		return err
	}
	return s.validateSubresources()
}

func (s ResourceSpec) validateCategory() error {
	switch s.Category {
	case handler.CostCategoryStandard:
		if s.Standard == nil || s.Standard.CostFunc == nil {
			return fmt.Errorf("resource spec %q: standard pricing function is required", s.Type)
		}
	case handler.CostCategoryFixed:
		if s.Fixed == nil || s.Fixed.CostFunc == nil {
			return fmt.Errorf("resource spec %q: fixed pricing function is required", s.Type)
		}
	case handler.CostCategoryUsageBased:
		if s.Usage == nil || s.Usage.EstimateFunc == nil {
			return fmt.Errorf("resource spec %q: usage pricing function is required", s.Type)
		}
	default:
		return fmt.Errorf("resource spec %q: unsupported category %v", s.Type, s.Category)
	}

	return nil
}

func (s ResourceSpec) validateLookup() error {
	if s.Lookup != nil && s.Lookup.BuildFunc == nil {
		return fmt.Errorf("resource spec %q: lookup spec requires build function", s.Type)
	}

	return nil
}

func (s ResourceSpec) validateDescribe() error {
	if s.Describe == nil {
		return nil
	}
	if s.Describe.BuildFunc == nil && len(s.Describe.Fields) == 0 {
		return fmt.Errorf("resource spec %q: describe spec requires fields or build function", s.Type)
	}

	for _, field := range s.Describe.Fields {
		if field.Key == "" {
			return fmt.Errorf("resource spec %q: describe field key is required", s.Type)
		}
		if field.Value == nil {
			return fmt.Errorf("resource spec %q: describe field %q requires value resolver", s.Type, field.Key)
		}
	}

	return nil
}

func (s ResourceSpec) validateSubresources() error {
	if s.Subresources != nil && s.Subresources.BuildFunc == nil {
		return fmt.Errorf("resource spec %q: subresource spec requires build function", s.Type)
	}

	return nil
}

// Const returns a constant value resolver.
func Const(value string) ValueFunc {
	return func(_ *pricing.Price, _ map[string]any) (string, bool) {
		return value, true
	}
}

// StringAttr resolves a string attribute.
func StringAttr(key string) ValueFunc {
	return func(_ *pricing.Price, attrs map[string]any) (string, bool) {
		value := handler.GetStringAttr(attrs, key)
		if value == "" {
			return "", false
		}
		return value, true
	}
}

// IntAttr resolves an integer attribute as a string.
func IntAttr(key string) ValueFunc {
	return func(_ *pricing.Price, attrs map[string]any) (string, bool) {
		value := handler.GetIntAttr(attrs, key)
		if value == 0 {
			return "", false
		}
		return strconv.Itoa(value), true
	}
}

// Describe builds the final detail map for one resource instance.
func (s *DescribeSpec) Describe(price *pricing.Price, attrs map[string]any) map[string]string {
	if s == nil {
		return nil
	}
	if s.BuildFunc != nil {
		return s.BuildFunc(price, attrs)
	}

	details := make(map[string]string, len(s.Fields))
	for _, field := range s.Fields {
		value, ok := field.Value(price, attrs)
		if !ok {
			continue
		}
		if field.OmitEmpty && value == "" {
			continue
		}
		details[field.Key] = value
	}
	if len(details) == 0 {
		return nil
	}
	return details
}

// Compile turns a typed resource spec into the canonical runtime definition.
func Compile(spec ResourceSpec) (resourcedef.Definition, error) {
	if err := spec.Validate(); err != nil {
		return resourcedef.Definition{}, err
	}

	def := resourcedef.Definition{
		Type:     spec.Type,
		Category: spec.Category,
	}
	if spec.Lookup != nil {
		def.Lookup = func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
			return spec.Lookup.BuildFunc(region, attrs)
		}
	}
	if spec.Describe != nil {
		def.Describe = spec.Describe.Describe
	}
	if spec.Standard != nil {
		def.StandardCost = func(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
			return spec.Standard.CostFunc(price, index, region, attrs)
		}
	}
	if spec.Fixed != nil {
		def.FixedCost = func(region string, attrs map[string]any) (hourly, monthly float64) {
			return spec.Fixed.CostFunc(region, attrs)
		}
	}
	if spec.Usage != nil {
		def.UsageCost = func(region string, attrs map[string]any) model.UsageCostEstimate {
			return spec.Usage.EstimateFunc(region, attrs)
		}
	}
	if spec.Subresources != nil {
		def.Subresources = func(attrs map[string]any) []handler.SubResource {
			return spec.Subresources.BuildFunc(attrs)
		}
	}

	return def, nil
}

// MustCompile turns a spec into a runtime definition and panics on invalid configuration.
func MustCompile(spec ResourceSpec) resourcedef.Definition {
	def, err := Compile(spec)
	if err != nil {
		panic(err)
	}
	return def
}
