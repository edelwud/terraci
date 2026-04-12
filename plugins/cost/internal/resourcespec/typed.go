package resourcespec

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

// TypedSpec is a resource spec where all callbacks receive pre-parsed typed attributes.
// A is the resource's own attrs struct (e.g., instanceAttrs, ebsVolumeAttrs).
// It compiles down to the same resourcedef.Definition via CompileTyped.
type TypedSpec[A any] struct {
	Type     resourcedef.ResourceType
	Category resourcedef.CostCategory
	Parse    func(attrs map[string]any) A

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
func ParseNoAttrs(map[string]any) NoAttrs {
	return NoAttrs{}
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
// The Parse function wraps each callback so resource logic receives pre-parsed attributes.
func CompileTyped[A any](spec TypedSpec[A]) (resourcedef.Definition, error) {
	if err := spec.Validate(); err != nil {
		return resourcedef.Definition{}, err
	}

	parse := spec.Parse
	untyped := ResourceSpec{
		Type:     spec.Type,
		Category: spec.Category,
	}

	if spec.Lookup != nil {
		fn := spec.Lookup.BuildFunc
		untyped.Lookup = &LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				return fn(region, parse(attrs))
			},
		}
	}
	if spec.Describe != nil {
		fn := spec.Describe.BuildFunc
		untyped.Describe = &DescribeSpec{
			BuildFunc: func(price *pricing.Price, attrs map[string]any) map[string]string {
				return fn(price, parse(attrs))
			},
		}
	}
	if spec.Standard != nil {
		fn := spec.Standard.CostFunc
		untyped.Standard = &StandardPricingSpec{
			CostFunc: func(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
				return fn(price, index, region, parse(attrs))
			},
		}
	}
	if spec.Fixed != nil {
		fn := spec.Fixed.CostFunc
		untyped.Fixed = &FixedPricingSpec{
			CostFunc: func(region string, attrs map[string]any) (hourly, monthly float64) {
				return fn(region, parse(attrs))
			},
		}
	}
	if spec.Usage != nil {
		fn := spec.Usage.EstimateFunc
		untyped.Usage = &UsagePricingSpec{
			EstimateFunc: func(region string, attrs map[string]any) model.UsageCostEstimate {
				return fn(region, parse(attrs))
			},
		}
	}
	if spec.Subresources != nil {
		fn := spec.Subresources.BuildFunc
		untyped.Subresources = &SubresourceSpec{
			BuildFunc: func(attrs map[string]any) []resourcedef.SubResource {
				return fn(parse(attrs))
			},
		}
	}

	return Compile(untyped)
}

// MustCompileTyped turns a typed spec into a runtime definition and panics on invalid configuration.
func MustCompileTyped[A any](spec TypedSpec[A]) resourcedef.Definition {
	def, err := CompileTyped(spec)
	if err != nil {
		panic(err)
	}
	return def
}
