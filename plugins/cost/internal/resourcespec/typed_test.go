package resourcespec_test

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

type testAttrs struct {
	Size string
	IOPS int
}

func parseTestAttrs(attrs map[string]any) testAttrs {
	return testAttrs{
		Size: costutil.GetStringAttr(attrs, "size"),
		IOPS: costutil.GetIntAttr(attrs, "iops"),
	}
}

func TestCompileTyped_StandardSpec(t *testing.T) {
	t.Parallel()

	spec := resourcespec.TypedSpec[testAttrs]{
		Type:     "aws_test_typed_standard",
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseTestAttrs,
		Lookup: &resourcespec.TypedLookupSpec[testAttrs]{
			BuildFunc: func(_ string, p testAttrs) (*pricing.PriceLookup, error) {
				return &pricing.PriceLookup{
					ServiceID:     pricing.ServiceID{Provider: "aws", Name: "test"},
					ProductFamily: "Test",
					Attributes:    map[string]string{"size": p.Size},
				}, nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[testAttrs]{
			BuildFunc: func(_ *pricing.Price, p testAttrs) map[string]string {
				return map[string]string{"size": p.Size}
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[testAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ testAttrs) (hourly, monthly float64) {
				return costutil.HourlyCost(price.OnDemandUSD)
			},
		},
	}

	def := mustCompileTyped(t, spec)

	lookup, err := def.BuildLookup("us-east-1", map[string]any{"size": "large"})
	if err != nil {
		t.Fatalf("BuildLookup() error = %v", err)
	}
	if lookup.Attributes["size"] != "large" {
		t.Fatalf("size = %q, want %q", lookup.Attributes["size"], "large")
	}

	details := def.DescribeResource(nil, map[string]any{"size": "large"})
	if details["size"] != "large" {
		t.Fatalf("Describe()[size] = %q, want %q", details["size"], "large")
	}

	_, monthly, ok := def.CalculateStandardCost(&pricing.Price{OnDemandUSD: 0.10}, nil, "us-east-1", nil)
	if !ok {
		t.Fatal("standard cost should be available")
	}
	if monthly != 0.10*costutil.HoursPerMonth {
		t.Fatalf("monthly = %.2f, want %.2f", monthly, 0.10*costutil.HoursPerMonth)
	}
}

func TestCompileTyped_FixedSpec(t *testing.T) {
	t.Parallel()

	spec := resourcespec.TypedSpec[testAttrs]{
		Type:     "aws_test_typed_fixed",
		Category: resourcedef.CostCategoryFixed,
		Parse:    parseTestAttrs,
		Fixed: &resourcespec.TypedFixedPricingSpec[testAttrs]{
			CostFunc: func(_ string, p testAttrs) (hourly, monthly float64) {
				if p.Size == "large" {
					return costutil.FixedMonthlyCost(5.0)
				}
				return costutil.FixedMonthlyCost(2.5)
			},
		},
	}

	def := mustCompileTyped(t, spec)

	_, monthly, ok := def.CalculateFixedCost("", map[string]any{"size": "large"})
	if !ok {
		t.Fatal("fixed cost should be available")
	}
	if monthly != 5.0 {
		t.Fatalf("monthly = %.2f, want 5.0", monthly)
	}
}

func TestCompileTyped_UsageSpec(t *testing.T) {
	t.Parallel()

	spec := resourcespec.TypedSpec[testAttrs]{
		Type:     "aws_test_typed_usage",
		Category: resourcedef.CostCategoryUsageBased,
		Parse:    parseTestAttrs,
		Usage: &resourcespec.TypedUsagePricingSpec[testAttrs]{
			EstimateFunc: func(_ string, p testAttrs) model.UsageCostEstimate {
				if p.IOPS > 0 {
					return model.UsageCostEstimate{
						Status:      model.ResourceEstimateStatusUsageEstimated,
						MonthlyCost: float64(p.IOPS) * 0.01,
					}
				}
				return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
			},
		},
	}

	def := mustCompileTyped(t, spec)

	estimate, ok := def.CalculateUsageCost("", map[string]any{"iops": 1000})
	if !ok {
		t.Fatal("usage cost should be available")
	}
	if estimate.MonthlyCost != 10.0 {
		t.Fatalf("monthly = %.2f, want 10.0", estimate.MonthlyCost)
	}
}

func TestFixedMonthlyNoAttrsSpec(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(resourcespec.FixedMonthlyNoAttrsSpec("aws_test_fixed_noattrs", 12.0))

	_, monthly, ok := def.CalculateFixedCost("", nil)
	if !ok {
		t.Fatal("fixed cost should be available")
	}
	if monthly != 12.0 {
		t.Fatalf("monthly = %.2f, want 12.0", monthly)
	}

	lookup, err := def.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup() error = %v", err)
	}
	if lookup != nil {
		t.Fatalf("BuildLookup() = %#v, want nil", lookup)
	}
}

func TestUsageUnknownNoAttrsSpec(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(resourcespec.UsageUnknownNoAttrsSpec("aws_test_usage_noattrs"))

	estimate, ok := def.CalculateUsageCost("", nil)
	if !ok {
		t.Fatal("usage cost should be available")
	}
	if estimate.Status != model.ResourceEstimateStatusUsageUnknown {
		t.Fatalf("status = %q, want %q", estimate.Status, model.ResourceEstimateStatusUsageUnknown)
	}

	if got := def.DescribeResource(nil, nil); got != nil {
		t.Fatalf("DescribeResource() = %#v, want nil", got)
	}
}

func TestCompileTyped_Subresources(t *testing.T) {
	t.Parallel()

	spec := resourcespec.TypedSpec[testAttrs]{
		Type:     "aws_test_typed_compound",
		Category: resourcedef.CostCategoryFixed,
		Parse:    parseTestAttrs,
		Fixed: &resourcespec.TypedFixedPricingSpec[testAttrs]{
			CostFunc: func(_ string, _ testAttrs) (hourly, monthly float64) { return 0, 0 },
		},
		Subresources: &resourcespec.TypedSubresourceSpec[testAttrs]{
			BuildFunc: func(p testAttrs) []resourcedef.SubResource {
				return []resourcedef.SubResource{{
					Suffix: "/disk",
					Type:   "aws_ebs_volume",
					Attrs:  map[string]any{"size": p.Size},
				}}
			},
		},
	}

	def := mustCompileTyped(t, spec)

	subs := def.BuildSubresources(map[string]any{"size": "100"})
	if len(subs) != 1 {
		t.Fatalf("subresources = %d, want 1", len(subs))
	}
	if subs[0].Attrs["size"] != "100" {
		t.Fatalf("sub attrs[size] = %v, want %q", subs[0].Attrs["size"], "100")
	}
}

func TestCompileTyped_NilParse(t *testing.T) {
	t.Parallel()

	_, err := resourcespec.CompileTyped(resourcespec.TypedSpec[testAttrs]{
		Type:     "aws_no_parse",
		Category: resourcedef.CostCategoryFixed,
		Fixed: &resourcespec.TypedFixedPricingSpec[testAttrs]{
			CostFunc: func(_ string, _ testAttrs) (hourly, monthly float64) { return 0, 0 },
		},
	})
	if err == nil {
		t.Fatal("expected error for nil parse function")
	}
}

func TestCompileTyped_InvalidCategory(t *testing.T) {
	t.Parallel()

	_, err := resourcespec.CompileTyped(resourcespec.TypedSpec[testAttrs]{
		Type:     "aws_invalid_typed",
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseTestAttrs,
	})
	if err == nil {
		t.Fatal("expected error for missing standard cost func")
	}
}

func TestCompileTyped_NilLookupBuildFunc(t *testing.T) {
	t.Parallel()

	_, err := resourcespec.CompileTyped(resourcespec.TypedSpec[testAttrs]{
		Type:     "aws_invalid_lookup",
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseTestAttrs,
		Lookup:   &resourcespec.TypedLookupSpec[testAttrs]{},
		Standard: &resourcespec.TypedStandardPricingSpec[testAttrs]{
			CostFunc: func(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ testAttrs) (hourly, monthly float64) {
				return 0, 0
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for nil lookup build function")
	}
}

func TestCompileTyped_NilDescribeBuildFunc(t *testing.T) {
	t.Parallel()

	_, err := resourcespec.CompileTyped(resourcespec.TypedSpec[testAttrs]{
		Type:     "aws_invalid_describe",
		Category: resourcedef.CostCategoryFixed,
		Parse:    parseTestAttrs,
		Describe: &resourcespec.TypedDescribeSpec[testAttrs]{},
		Fixed: &resourcespec.TypedFixedPricingSpec[testAttrs]{
			CostFunc: func(_ string, _ testAttrs) (hourly, monthly float64) { return 0, 0 },
		},
	})
	if err == nil {
		t.Fatal("expected error for nil describe build function")
	}
}

func TestCompileTyped_NilUsageEstimateFunc(t *testing.T) {
	t.Parallel()

	_, err := resourcespec.CompileTyped(resourcespec.TypedSpec[testAttrs]{
		Type:     "aws_invalid_usage",
		Category: resourcedef.CostCategoryUsageBased,
		Parse:    parseTestAttrs,
		Usage:    &resourcespec.TypedUsagePricingSpec[testAttrs]{},
	})
	if err == nil {
		t.Fatal("expected error for nil usage estimate function")
	}
}

func TestMustCompileTyped_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	resourcespec.MustCompileTyped(resourcespec.TypedSpec[testAttrs]{
		Type: "aws_panic",
	})
}

func mustCompileTyped[A any](t *testing.T, spec resourcespec.TypedSpec[A]) resourcedef.Definition {
	t.Helper()

	def, err := resourcespec.CompileTyped(spec)
	if err != nil {
		t.Fatalf("CompileTyped() error = %v", err)
	}
	return def
}
