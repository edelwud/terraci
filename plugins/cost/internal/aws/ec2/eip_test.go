package ec2

import (
	"testing"

	aws "github.com/edelwud/terraci/plugins/cost/internal/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestEIPHandler_Category(t *testing.T) {
	t.Parallel()

	h := &EIPHandler{}
	if h.Category() != aws.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestEIPHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &EIPHandler{}

	// Without instance → attached=false
	result := h.Describe(nil, map[string]any{})
	if result["attached"] != "false" {
		t.Errorf("Describe()[attached] = %q, want %q", result["attached"], "false")
	}

	// With instance → attached=true
	result = h.Describe(nil, map[string]any{"instance": "i-12345"})
	if result["attached"] != "true" {
		t.Errorf("Describe()[attached] = %q, want %q", result["attached"], "true")
	}
}

func TestEIPHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &EIPHandler{}
	if h.ServiceCode() != pricing.ServiceVPC {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceVPC)
	}
}

func TestEIPHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &EIPHandler{}

	// Without instance (idle)
	lookup, err := h.BuildLookup("us-east-1", map[string]any{})
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup.Attributes["group"] != "VPCPublicIPv4Address" {
		t.Errorf("group = %q, want %q", lookup.Attributes["group"], "VPCPublicIPv4Address")
	}

	// With instance (in-use)
	lookup, err = h.BuildLookup("us-east-1", map[string]any{"instance": "i-12345"})
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup.Attributes["group"] != "VPCPublicIPv4Address" {
		t.Errorf("group = %q, want %q", lookup.Attributes["group"], "VPCPublicIPv4Address")
	}
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
