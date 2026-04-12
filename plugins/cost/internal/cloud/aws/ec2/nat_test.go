package ec2

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestNATHandler_Category(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryStandard
	handlertest.RunContractSuite(t, resourcespec.MustHandler(NATSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))), handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
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
		DescribeCases: []handlertest.DescribeCase{
			{
				Name: "empty describe",
				Assert: func(tb testing.TB, result map[string]string) {
					tb.Helper()
					if len(result) != 0 {
						tb.Errorf("Describe() = %v, want empty map", result)
					}
				},
			},
		},
	})
}

func TestNATHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(NATSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.StandardCostHandler)
	if !ok {
		t.Fatal("handler should implement StandardCostHandler")
	}

	// With price from lookup
	price := &pricing.Price{
		OnDemandUSD: 0.045,
	}

	hourly, monthly := h.CalculateCost(price, nil, "", nil)

	if hourly != 0.045 {
		t.Errorf("hourly = %v, want %v", hourly, 0.045)
	}

	expectedMonthly := 0.045 * 730
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}

	// Without price (fallback)
	hourly, _ = h.CalculateCost(&pricing.Price{OnDemandUSD: 0}, nil, "", nil)
	if hourly != 0.045 {
		t.Errorf("fallback hourly = %v, want %v", hourly, 0.045)
	}
}
