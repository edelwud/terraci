package ec2

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/contracttest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestNATHandler_Category(t *testing.T) {
	t.Parallel()

	category := resourcedef.CostCategoryStandard
	contracttest.RunContractSuite(t, resourcespec.MustCompileTyped(NATSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))), contracttest.ContractSuite{
		Category: &category,
		LookupCases: []contracttest.LookupCase{
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
		DescribeCases: []contracttest.DescribeCase{
			{
				Name: "default attrs",
				Assert: func(tb testing.TB, result map[string]string) {
					tb.Helper()
					if result["connectivity"] != DefaultNATConnectivityType {
						tb.Errorf("connectivity = %q, want %q", result["connectivity"], DefaultNATConnectivityType)
					}
				},
			},
			{
				Name:  "with public ip",
				Attrs: map[string]any{"public_ip": "203.0.113.5", "connectivity_type": DefaultNATConnectivityType},
				Assert: func(tb testing.TB, result map[string]string) {
					tb.Helper()
					if result["public_ip"] != "203.0.113.5" {
						tb.Errorf("public_ip = %q, want %q", result["public_ip"], "203.0.113.5")
					}
					if result["connectivity"] != DefaultNATConnectivityType {
						tb.Errorf("connectivity = %q, want %q", result["connectivity"], DefaultNATConnectivityType)
					}
				},
			},
			{
				Name:  "private nat gateway",
				Attrs: map[string]any{"connectivity_type": "private"},
				Assert: func(tb testing.TB, result map[string]string) {
					tb.Helper()
					if result["connectivity"] != "private" {
						tb.Errorf("connectivity = %q, want %q", result["connectivity"], "private")
					}
					if _, ok := result["public_ip"]; ok {
						tb.Errorf("public_ip should not be present for private NAT")
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
