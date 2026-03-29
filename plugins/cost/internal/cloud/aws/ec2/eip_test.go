package ec2

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestEIPHandler_Category(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryStandard
	handlertest.RunContractSuite(t, &EIPHandler{}, handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
			{
				Name:   "idle address",
				Region: "us-east-1",
				Attrs:  map[string]any{},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["group"] != "VPCPublicIPv4Address" {
						tb.Errorf("group = %q, want %q", lookup.Attributes["group"], "VPCPublicIPv4Address")
					}
				},
			},
			{
				Name:   "attached address",
				Region: "us-east-1",
				Attrs:  map[string]any{"instance": "i-12345"},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["group"] != "VPCPublicIPv4Address" {
						tb.Errorf("group = %q, want %q", lookup.Attributes["group"], "VPCPublicIPv4Address")
					}
				},
			},
		},
		DescribeCases: []handlertest.DescribeCase{
			{
				Name:     "idle address",
				Attrs:    map[string]any{},
				WantKeys: map[string]string{"attached": "false"},
			},
			{
				Name:     "attached address",
				Attrs:    map[string]any{"instance": "i-12345"},
				WantKeys: map[string]string{"attached": "true"},
			},
		},
	})
}

func TestEIPHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &EIPHandler{}

	// With price from lookup
	price := &pricing.Price{OnDemandUSD: 0.005}
	hourly, monthly := h.CalculateCost(price, nil, "", nil)
	if hourly != 0.005 {
		t.Errorf("hourly = %v, want %v", hourly, 0.005)
	}
	expectedMonthly := 0.005 * 730
	if monthly != expectedMonthly {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}

	// Fallback when price is zero
	hourly, _ = h.CalculateCost(&pricing.Price{OnDemandUSD: 0}, nil, "", nil)
	if hourly != DefaultEIPHourlyCost {
		t.Errorf("fallback hourly = %v, want %v", hourly, DefaultEIPHourlyCost)
	}

	// Fallback when price is nil
	hourly, _ = h.CalculateCost(nil, nil, "", nil)
	if hourly != DefaultEIPHourlyCost {
		t.Errorf("nil price hourly = %v, want %v", hourly, DefaultEIPHourlyCost)
	}
}
