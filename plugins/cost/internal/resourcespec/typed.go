package resourcespec

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

// TypedSpec is the canonical declaration for one resource's pricing behavior.
// A is the resource's own parsed attrs struct (for example, instanceAttrs).
type TypedSpec[A any] struct {
	Type     resourcedef.ResourceType
	Category resourcedef.CostCategory
	Parse    func(attrs resourcedef.RawAttrs) (A, error)

	Lookup       *TypedLookupSpec[A]
	Describe     *TypedDescribeSpec[A]
	Standard     *TypedStandardPricingSpec[A]
	Fixed        *TypedFixedPricingSpec[A]
	Usage        *TypedUsagePricingSpec[A]
	Subresources *TypedSubresourceSpec[A]
}

// TypedLookupSpec declares pricing lookup behavior with pre-parsed attributes.
type TypedLookupSpec[A any] struct {
	BuildFunc func(region string, parsed A) (*pricing.PriceLookup, error)
}

// TypedDescribeSpec declares resource description behavior with pre-parsed attributes.
type TypedDescribeSpec[A any] struct {
	BuildFunc func(price *pricing.Price, parsed A) map[string]string
}

// TypedStandardPricingSpec declares standard pricing behavior with pre-parsed attributes.
type TypedStandardPricingSpec[A any] struct {
	CostFunc func(price *pricing.Price, index *pricing.PriceIndex, region string, parsed A) (hourly, monthly float64)
}

// TypedFixedPricingSpec declares fixed pricing behavior with pre-parsed attributes.
type TypedFixedPricingSpec[A any] struct {
	CostFunc func(region string, parsed A) (hourly, monthly float64)
}

// TypedUsagePricingSpec declares usage-based pricing behavior with pre-parsed attributes.
type TypedUsagePricingSpec[A any] struct {
	EstimateFunc func(region string, parsed A) model.UsageCostEstimate
}

// TypedSubresourceSpec declares subresource expansion behavior with pre-parsed attributes.
type TypedSubresourceSpec[A any] struct {
	BuildFunc func(parsed A) []resourcedef.SubResource
}

// NoAttrs is a convenience type for resources that do not need parsed attributes.
type NoAttrs struct{}

// ParseNoAttrs ignores raw attributes and returns an empty typed payload.
func ParseNoAttrs(resourcedef.RawAttrs) (NoAttrs, error) {
	return NoAttrs{}, nil
}

// NoAttrsSpec creates a typed spec for resources that do not need parsed attributes.
func NoAttrsSpec(resourceType resourcedef.ResourceType, category resourcedef.CostCategory) TypedSpec[NoAttrs] {
	return TypedSpec[NoAttrs]{
		Type:     resourceType,
		Category: category,
		Parse:    ParseNoAttrs,
	}
}

// FixedMonthlyNoAttrsSpec creates a fixed-price no-attrs spec with a constant monthly cost.
func FixedMonthlyNoAttrsSpec(resourceType resourcedef.ResourceType, monthlyCost float64) TypedSpec[NoAttrs] {
	spec := NoAttrsSpec(resourceType, resourcedef.CostCategoryFixed)
	spec.Fixed = &TypedFixedPricingSpec[NoAttrs]{
		CostFunc: func(_ string, _ NoAttrs) (hourly, monthly float64) {
			return monthlyCost / costutil.HoursPerMonth, monthlyCost
		},
	}
	return spec
}

// UsageUnknownNoAttrsSpec creates a no-attrs usage-based spec with unknown plan-time usage.
func UsageUnknownNoAttrsSpec(resourceType resourcedef.ResourceType) TypedSpec[NoAttrs] {
	spec := NoAttrsSpec(resourceType, resourcedef.CostCategoryUsageBased)
	spec.Usage = &TypedUsagePricingSpec[NoAttrs]{
		EstimateFunc: func(_ string, _ NoAttrs) model.UsageCostEstimate {
			return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
		},
	}
	return spec
}

// Validate ensures the typed spec is internally consistent before compilation.
func (s TypedSpec[A]) Validate() error {
	if s.Type == "" {
		return errors.New("typed spec: type is required")
	}
	if s.Parse == nil {
		return fmt.Errorf("typed spec %q: parse function is required", s.Type)
	}

	switch s.Category {
	case resourcedef.CostCategoryStandard:
		if s.Standard == nil || s.Standard.CostFunc == nil {
			return fmt.Errorf("typed spec %q: standard pricing function is required", s.Type)
		}
	case resourcedef.CostCategoryFixed:
		if s.Fixed == nil || s.Fixed.CostFunc == nil {
			return fmt.Errorf("typed spec %q: fixed pricing function is required", s.Type)
		}
	case resourcedef.CostCategoryUsageBased:
		if s.Usage == nil || s.Usage.EstimateFunc == nil {
			return fmt.Errorf("typed spec %q: usage pricing function is required", s.Type)
		}
	default:
		return fmt.Errorf("typed spec %q: unsupported category %v", s.Type, s.Category)
	}

	if s.Lookup != nil && s.Lookup.BuildFunc == nil {
		return fmt.Errorf("typed spec %q: lookup spec requires build function", s.Type)
	}
	if s.Describe != nil && s.Describe.BuildFunc == nil {
		return fmt.Errorf("typed spec %q: describe spec requires build function", s.Type)
	}
	if s.Subresources != nil && s.Subresources.BuildFunc == nil {
		return fmt.Errorf("typed spec %q: subresource spec requires build function", s.Type)
	}

	return nil
}

// CompileTyped turns a typed resource spec into the canonical runtime definition.
func CompileTyped[A any](spec TypedSpec[A]) (resourcedef.Definition, error) {
	if err := spec.Validate(); err != nil {
		return resourcedef.Definition{}, err
	}

	def := resourcedef.Definition{
		Type:     spec.Type,
		Category: spec.Category,
		Parse: func(attrs resourcedef.RawAttrs) (resourcedef.Attributes, error) {
			parsed, err := spec.Parse(attrs)
			if err != nil {
				return resourcedef.Attributes{}, err
			}
			return resourcedef.NewAttributes(parsed), nil
		},
	}

	if spec.Lookup != nil {
		fn := spec.Lookup.BuildFunc
		def.Lookup = func(region string, attrs resourcedef.Attributes) (*pricing.PriceLookup, error) {
			parsed, err := typedAttrs[A](attrs)
			if err != nil {
				return nil, err
			}
			return fn(region, parsed)
		}
	}
	if spec.Describe != nil {
		fn := spec.Describe.BuildFunc
		def.Describe = func(price *pricing.Price, attrs resourcedef.Attributes) map[string]string {
			parsed, err := typedAttrs[A](attrs)
			if err != nil {
				return nil
			}
			return fn(price, parsed)
		}
	}
	if spec.Standard != nil {
		fn := spec.Standard.CostFunc
		def.StandardCost = func(price *pricing.Price, index *pricing.PriceIndex, region string, attrs resourcedef.Attributes) (hourly, monthly float64) {
			parsed, err := typedAttrs[A](attrs)
			if err != nil {
				return 0, 0
			}
			return fn(price, index, region, parsed)
		}
	}
	if spec.Fixed != nil {
		fn := spec.Fixed.CostFunc
		def.FixedCost = func(region string, attrs resourcedef.Attributes) (hourly, monthly float64) {
			parsed, err := typedAttrs[A](attrs)
			if err != nil {
				return 0, 0
			}
			return fn(region, parsed)
		}
	}
	if spec.Usage != nil {
		fn := spec.Usage.EstimateFunc
		def.UsageCost = func(region string, attrs resourcedef.Attributes) model.UsageCostEstimate {
			parsed, err := typedAttrs[A](attrs)
			if err != nil {
				return model.UsageCostEstimate{
					Status: model.ResourceEstimateStatusFailed,
					Detail: err.Error(),
				}
			}
			return fn(region, parsed)
		}
	}
	if spec.Subresources != nil {
		fn := spec.Subresources.BuildFunc
		def.Subresources = func(attrs resourcedef.Attributes) []resourcedef.SubResource {
			parsed, err := typedAttrs[A](attrs)
			if err != nil {
				return nil
			}
			return fn(parsed)
		}
	}

	if err := def.Validate(); err != nil {
		return resourcedef.Definition{}, err
	}
	return def, nil
}

// MustCompileTyped turns a typed spec into a runtime definition and panics on invalid configuration.
func MustCompileTyped[A any](spec TypedSpec[A]) resourcedef.Definition {
	def, err := CompileTyped(spec)
	if err != nil {
		panic(err)
	}
	return def
}

func typedAttrs[A any](attrs resourcedef.Attributes) (A, error) {
	parsed, err := resourcedef.AttributesAs[A](attrs)
	if err != nil {
		return parsed, fmt.Errorf("typed resource spec: %w", err)
	}
	return parsed, nil
}
