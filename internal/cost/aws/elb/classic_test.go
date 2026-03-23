package elb

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestClassicHandler_Category(t *testing.T) {
	t.Parallel()

	h := &ClassicHandler{}
	if h.Category() != aws.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestClassicHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &ClassicHandler{}
	result := h.Describe(nil, nil)
	if result["type"] != "classic" {
		t.Errorf("Describe()[type] = %q, want %q", result["type"], "classic")
	}
}

func TestClassicHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &ClassicHandler{}
	if h.ServiceCode() != pricing.ServiceELB {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceELB)
	}
}

func TestClassicHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &ClassicHandler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}

	if lookup.ProductFamily != "Load Balancer" {
		t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Load Balancer")
	}
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
