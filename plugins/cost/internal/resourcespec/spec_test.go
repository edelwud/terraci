package resourcespec_test

import (
	"errors"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestCompile_StandardSpec(t *testing.T) {
	t.Parallel()

	spec := resourcespec.ResourceSpec{
		Type:     "aws_test_standard",
		Category: handler.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				return &pricing.PriceLookup{
					ServiceID:     pricing.ServiceID{Provider: "aws", Name: "amazon_test"},
					ProductFamily: "Test",
					Attributes: map[string]string{
						"region": region,
						"size":   attrs["size"].(string),
					},
				}, nil
			},
		},
		Describe: &resourcespec.DescribeSpec{
			Fields: []resourcespec.DescribeField{
				{Key: "size", Value: resourcespec.StringAttr("size")},
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
				return price.OnDemandUSD, price.OnDemandUSD * 730
			},
		},
	}

	def := mustCompile(t, spec)
	lookup, err := def.BuildLookup("us-east-1", map[string]any{"size": "small"})
	if err != nil {
		t.Fatalf("BuildLookup() error = %v", err)
	}
	if lookup == nil {
		t.Fatal("BuildLookup() returned nil")
	}
	if lookup.Attributes["size"] != "small" {
		t.Fatalf("size = %q, want %q", lookup.Attributes["size"], "small")
	}

	_, monthly, ok := def.CalculateStandardCost(&pricing.Price{OnDemandUSD: 1.5}, nil, "us-east-1", nil)
	if !ok {
		t.Fatal("definition should expose standard cost behavior")
	}
	if monthly != 1095 {
		t.Fatalf("monthly = %.2f, want 1095", monthly)
	}

	if got := def.DescribeResource(nil, map[string]any{"size": "small"}); got["size"] != "small" {
		t.Fatalf("Describe()[size] = %q, want %q", got["size"], "small")
	}
}

func TestCompile_FixedSpec(t *testing.T) {
	t.Parallel()

	def := mustCompile(t, resourcespec.ResourceSpec{
		Type:     "aws_test_fixed",
		Category: handler.CostCategoryFixed,
		Fixed: &resourcespec.FixedPricingSpec{
			CostFunc: func(_ string, _ map[string]any) (hourly, monthly float64) {
				return handler.FixedMonthlyCost(2.5)
			},
		},
	})

	_, monthly, ok := def.CalculateFixedCost("", nil)
	if !ok {
		t.Fatal("definition should expose fixed cost behavior")
	}
	if monthly != 2.5 {
		t.Fatalf("monthly = %.2f, want 2.5", monthly)
	}
}

func TestCompile_UsageSpec(t *testing.T) {
	t.Parallel()

	spec := resourcespec.ResourceSpec{
		Type:     "aws_test_usage",
		Category: handler.CostCategoryUsageBased,
		Usage: &resourcespec.UsagePricingSpec{
			EstimateFunc: func(_ string, attrs map[string]any) model.UsageCostEstimate {
				if attrs["estimated"] == true {
					return model.UsageCostEstimate{
						HourlyCost:  1,
						MonthlyCost: 730,
						Status:      model.ResourceEstimateStatusUsageEstimated,
					}
				}
				return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
			},
		},
	}

	def := mustCompile(t, spec)
	estimate, ok := def.CalculateUsageCost("", map[string]any{"estimated": true})
	if !ok {
		t.Fatal("definition should expose usage cost behavior")
	}
	if estimate.Status != model.ResourceEstimateStatusUsageEstimated {
		t.Fatalf("status = %q, want %q", estimate.Status, model.ResourceEstimateStatusUsageEstimated)
	}
	if got, _ := def.CalculateUsageCost("", nil); got.Status != model.ResourceEstimateStatusUsageUnknown {
		t.Fatalf("status = %q, want %q", got.Status, model.ResourceEstimateStatusUsageUnknown)
	}
}

func TestDescribeSpec_OmitEmpty(t *testing.T) {
	t.Parallel()

	describe := &resourcespec.DescribeSpec{
		Fields: []resourcespec.DescribeField{
			{Key: "name", Value: resourcespec.StringAttr("name"), OmitEmpty: true},
			{Key: "kind", Value: resourcespec.Const("bucket")},
		},
	}

	got := describe.Describe(nil, map[string]any{})
	if _, ok := got["name"]; ok {
		t.Fatal("name should be omitted")
	}
	if got["kind"] != "bucket" {
		t.Fatalf("kind = %q, want %q", got["kind"], "bucket")
	}
}

func TestCompile_Subresources(t *testing.T) {
	t.Parallel()

	def := mustCompile(t, resourcespec.ResourceSpec{
		Type:     "aws_test_compound",
		Category: handler.CostCategoryFixed,
		Fixed: &resourcespec.FixedPricingSpec{
			CostFunc: func(_ string, _ map[string]any) (hourly, monthly float64) { return 0, 0 },
		},
		Subresources: &resourcespec.SubresourceSpec{
			BuildFunc: func(_ map[string]any) []handler.SubResource {
				return []handler.SubResource{{
					Suffix: "/child",
					Type:   "aws_child",
					Attrs:  map[string]any{"size": 10},
				}}
			},
		},
	})

	subs := def.BuildSubresources(nil)
	if len(subs) != 1 {
		t.Fatalf("subresources = %d, want 1", len(subs))
	}
	if subs[0].Suffix != "/child" {
		t.Fatalf("suffix = %q, want %q", subs[0].Suffix, "/child")
	}
}

func TestCompile_InvalidSpec(t *testing.T) {
	t.Parallel()

	_, err := resourcespec.Compile(resourcespec.ResourceSpec{
		Type:     "aws_invalid",
		Category: handler.CostCategoryStandard,
	})
	if err == nil {
		t.Fatal("expected error for invalid spec")
	}
}

func TestDescribeSpec_BuildFunc(t *testing.T) {
	t.Parallel()

	describe := &resourcespec.DescribeSpec{
		BuildFunc: func(_ *pricing.Price, _ map[string]any) map[string]string {
			return nil
		},
	}
	if got := describe.Describe(nil, nil); got != nil {
		t.Fatalf("Describe() = %#v, want nil", got)
	}
}

func TestCompile_LookupError(t *testing.T) {
	t.Parallel()

	def := mustCompile(t, resourcespec.ResourceSpec{
		Type:     "aws_lookup_error",
		Category: handler.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
				return nil, errors.New("boom")
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
				return 0, 0
			},
		},
	})

	if _, err := def.BuildLookup("", nil); err == nil {
		t.Fatal("expected lookup error")
	}
}

func mustCompile(t *testing.T, spec resourcespec.ResourceSpec) resourcedef.Definition {
	t.Helper()

	def, err := resourcespec.Compile(spec)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return def
}
