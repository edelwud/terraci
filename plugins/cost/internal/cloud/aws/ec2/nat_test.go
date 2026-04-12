package ec2

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/definitiontest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestNATHandler_Category(t *testing.T) {
	t.Parallel()

	category := resourcedef.CostCategoryStandard
	definitiontest.RunContractSuite(t, resourcespec.MustCompileTyped(NATSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))), definitiontest.ContractSuite{
		Category: &category,
		LookupCases: []definitiontest.LookupCase{
			{
				Name:   "default lookup",
				Region: "us-east-1",
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["group"] != "NGW:NatGateway" {
						tb.Errorf("group = %q, want %q", lookup.Attributes["group"], "NGW:NatGateway")
					}
				},
			},
		},
		DescribeCases: []definitiontest.DescribeCase{
			{
				Name: "empty describe",
				Assert: func(tb testing.TB, result map[string]string) {
					tb.Helper()
					if len(result) != 0 {
						tb.Errorf("DescribeResource() = %v, want empty map", result)
					}
				},
			},
		},
	})
}

func TestNATHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(NATSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	// With price from lookup
	price := &pricing.Price{
		OnDemandUSD: 0.045,
	}

	hourly, monthly, ok := def.CalculateStandardCost(price, nil, "", nil)
	if !ok {
		t.Fatal("CalculateStandardCost should return ok=true")
	}

	if hourly != 0.045 {
		t.Errorf("hourly = %v, want %v", hourly, 0.045)
	}

	expectedMonthly := 0.045 * 730
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}

	// Without price (fallback)
	hourly, _, ok = def.CalculateStandardCost(&pricing.Price{OnDemandUSD: 0}, nil, "", nil)
	if !ok {
		t.Fatal("CalculateStandardCost should return ok=true")
	}
	if hourly != 0.045 {
		t.Errorf("fallback hourly = %v, want %v", hourly, 0.045)
	}
}
