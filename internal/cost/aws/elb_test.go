package aws

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestLBHandler_ServiceCode(t *testing.T) {
	h := &LBHandler{}
	if h.ServiceCode() != pricing.ServiceEC2 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceEC2)
	}
}

func TestLBHandler_BuildLookup(t *testing.T) {
	h := &LBHandler{}

	tests := []struct {
		name       string
		attrs      map[string]interface{}
		wantFamily string
	}{
		{
			name:       "default ALB",
			attrs:      map[string]interface{}{},
			wantFamily: "Load Balancer-Application",
		},
		{
			name: "explicit ALB",
			attrs: map[string]interface{}{
				"load_balancer_type": "application",
			},
			wantFamily: "Load Balancer-Application",
		},
		{
			name: "NLB",
			attrs: map[string]interface{}{
				"load_balancer_type": "network",
			},
			wantFamily: "Load Balancer-Network",
		},
		{
			name: "Gateway LB",
			attrs: map[string]interface{}{
				"load_balancer_type": "gateway",
			},
			wantFamily: "Load Balancer-Gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestLBHandler_CalculateCost(t *testing.T) {
	h := &LBHandler{}

	tests := []struct {
		name           string
		price          *pricing.Price
		attrs          map[string]interface{}
		expectedHourly float64
	}{
		{
			name:           "with price",
			price:          &pricing.Price{OnDemandUSD: 0.025},
			attrs:          map[string]interface{}{},
			expectedHourly: 0.025,
		},
		{
			name:           "fallback ALB",
			price:          &pricing.Price{OnDemandUSD: 0},
			attrs:          map[string]interface{}{},
			expectedHourly: 0.0225,
		},
		{
			name:  "fallback NLB",
			price: &pricing.Price{OnDemandUSD: 0},
			attrs: map[string]interface{}{
				"load_balancer_type": "network",
			},
			expectedHourly: 0.0225,
		},
		{
			name:  "fallback GWLB",
			price: &pricing.Price{OnDemandUSD: 0},
			attrs: map[string]interface{}{
				"load_balancer_type": "gateway",
			},
			expectedHourly: 0.0125,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hourly, _ := h.CalculateCost(tt.price, tt.attrs)

			if hourly != tt.expectedHourly {
				t.Errorf("hourly = %v, want %v", hourly, tt.expectedHourly)
			}
		})
	}
}

func TestClassicLBHandler_ServiceCode(t *testing.T) {
	h := &ClassicLBHandler{}
	if h.ServiceCode() != pricing.ServiceELB {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceELB)
	}
}

func TestClassicLBHandler_BuildLookup(t *testing.T) {
	h := &ClassicLBHandler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}

	if lookup.ProductFamily != "Load Balancer" {
		t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Load Balancer")
	}
}

func TestClassicLBHandler_CalculateCost(t *testing.T) {
	h := &ClassicLBHandler{}

	// With price
	price := &pricing.Price{OnDemandUSD: 0.03}
	hourly, monthly := h.CalculateCost(price, nil)
	if hourly != 0.03 {
		t.Errorf("hourly = %v, want %v", hourly, 0.03)
	}
	if monthly != 0.03*730 {
		t.Errorf("monthly = %v, want %v", monthly, 0.03*730)
	}

	// Fallback
	hourly, _ = h.CalculateCost(&pricing.Price{OnDemandUSD: 0}, nil)
	if hourly != 0.025 {
		t.Errorf("fallback hourly = %v, want %v", hourly, 0.025)
	}
}
