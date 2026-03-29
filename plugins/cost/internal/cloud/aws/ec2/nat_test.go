package ec2

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestNATHandler_Category(t *testing.T) {
	t.Parallel()

	h := &NATHandler{}
	if h.Category() != handler.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestNATHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &NATHandler{}
	result := h.Describe(nil, nil)
	if len(result) != 0 {
		t.Errorf("Describe() = %v, want empty map", result)
	}
}

func TestNATHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &NATHandler{}
	if h.ServiceCode() != awskit.MustService(awskit.ServiceKeyEC2) {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), awskit.MustService(awskit.ServiceKeyEC2))
	}
}

func TestNATHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &NATHandler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}

	if lookup.Attributes["group"] != "NGW:NatGateway" {
		t.Errorf("group = %q, want %q", lookup.Attributes["group"], "NGW:NatGateway")
	}
}

func TestNATHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &NATHandler{}

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
