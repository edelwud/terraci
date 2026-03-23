package elb

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestALBHandler_Category(t *testing.T) {
	t.Parallel()

	h := &ALBHandler{}
	if h.Category() != aws.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestALBHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &ALBHandler{}

	// Default → application
	result := h.Describe(nil, map[string]any{})
	if result["type"] != "application" {
		t.Errorf("Describe()[type] = %q, want %q", result["type"], "application")
	}

	// Explicit NLB
	result = h.Describe(nil, map[string]any{"load_balancer_type": "network"})
	if result["type"] != "network" {
		t.Errorf("Describe()[type] = %q, want %q", result["type"], "network")
	}
}

func TestALBHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &ALBHandler{}
	if h.ServiceCode() != pricing.ServiceEC2 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceEC2)
	}
}

func TestALBHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &ALBHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantFamily string
	}{
		{
			name:       "default ALB",
			attrs:      map[string]any{},
			wantFamily: "Load Balancer-Application",
		},
		{
			name: "explicit ALB",
			attrs: map[string]any{
				"load_balancer_type": "application",
			},
			wantFamily: "Load Balancer-Application",
		},
		{
			name: "NLB",
			attrs: map[string]any{
				"load_balancer_type": "network",
			},
			wantFamily: "Load Balancer-Network",
		},
		{
			name: "Gateway LB",
			attrs: map[string]any{
				"load_balancer_type": "gateway",
			},
			wantFamily: "Load Balancer-Gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookup, err := h.BuildLookup("us-east-1", tt.attrs)
			if err != nil {
				t.Fatalf("BuildLookup returned error: %v", err)
			}

			if lookup.ProductFamily != tt.wantFamily {
				t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, tt.wantFamily)
			}
		})
	}
}

func TestALBHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &ALBHandler{}

	tests := []struct {
		name           string
		price          *pricing.Price
		attrs          map[string]any
		expectedHourly float64
	}{
		{
			name:           "with price",
			price:          &pricing.Price{OnDemandUSD: 0.025},
			attrs:          map[string]any{},
			expectedHourly: 0.025,
		},
		{
			name:           "fallback ALB",
			price:          &pricing.Price{OnDemandUSD: 0},
			attrs:          map[string]any{},
			expectedHourly: 0.0225,
		},
		{
			name:  "fallback NLB",
			price: &pricing.Price{OnDemandUSD: 0},
			attrs: map[string]any{
				"load_balancer_type": "network",
			},
			expectedHourly: 0.0225,
		},
		{
			name:  "fallback GWLB",
			price: &pricing.Price{OnDemandUSD: 0},
			attrs: map[string]any{
				"load_balancer_type": "gateway",
			},
			expectedHourly: 0.0125,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hourly, _ := h.CalculateCost(tt.price, nil, "", tt.attrs)

			if hourly != tt.expectedHourly {
				t.Errorf("hourly = %v, want %v", hourly, tt.expectedHourly)
			}
		})
	}
}
