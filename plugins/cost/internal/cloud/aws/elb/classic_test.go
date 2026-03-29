package elb

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestClassicHandler_Category(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryStandard
	handlertest.RunContractSuite(t, &ClassicHandler{}, handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
			{
				Name:   "default lookup",
				Region: "us-east-1",
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.ProductFamily != "Load Balancer" {
						tb.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Load Balancer")
					}
				},
			},
		},
		DescribeCases: []handlertest.DescribeCase{
			{
				Name:     "default describe",
				WantKeys: map[string]string{"type": "classic"},
			},
		},
	})
}

func TestClassicHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &ClassicHandler{}

	// With price
	price := &pricing.Price{OnDemandUSD: 0.03}
	hourly, monthly := h.CalculateCost(price, nil, "", nil)
	if hourly != 0.03 {
		t.Errorf("hourly = %v, want %v", hourly, 0.03)
	}
	if monthly != 0.03*730 {
		t.Errorf("monthly = %v, want %v", monthly, 0.03*730)
	}

	// Fallback
	hourly, _ = h.CalculateCost(&pricing.Price{OnDemandUSD: 0}, nil, "", nil)
	if hourly != 0.025 {
		t.Errorf("fallback hourly = %v, want %v", hourly, 0.025)
	}
}
