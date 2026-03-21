package aws

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestEKSClusterHandler_ServiceCode(t *testing.T) {
	h := &EKSClusterHandler{}
	if h.ServiceCode() != pricing.ServiceEKS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceEKS)
	}
}

func TestEKSClusterHandler_BuildLookup_ReturnsNil(t *testing.T) {
	h := &EKSClusterHandler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	// EKS cluster uses fixed cost, no pricing API lookup needed
	if lookup != nil || err != nil {
		t.Errorf("BuildLookup should return nil, nil; got %v, %v", lookup, err)
	}
}

func TestEKSClusterHandler_CalculateCost_FixedPrice(t *testing.T) {
	h := &EKSClusterHandler{}

	hourly, monthly := h.CalculateCost(nil, nil)
	if hourly != DefaultEKSClusterHourlyCost {
		t.Errorf("hourly = %v, want %v", hourly, DefaultEKSClusterHourlyCost)
	}
	if monthly != DefaultEKSClusterHourlyCost*HoursPerMonth {
		t.Errorf("monthly = %v, want %v", monthly, DefaultEKSClusterHourlyCost*HoursPerMonth)
	}
}

func TestEKSNodeGroupHandler_ServiceCode(t *testing.T) {
	h := &EKSNodeGroupHandler{}
	if h.ServiceCode() != pricing.ServiceEC2 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceEC2)
	}
}

func TestEKSNodeGroupHandler_BuildLookup(t *testing.T) {
	h := &EKSNodeGroupHandler{}

	tests := []struct {
		name         string
		attrs        map[string]any
		wantInstance string
	}{
		{"with instance_types", map[string]any{"instance_types": []any{"m5.large"}}, "m5.large"},
		{"default", map[string]any{}, "t3.medium"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup, err := h.BuildLookup("us-east-1", tt.attrs)
			if err != nil {
				t.Fatalf("BuildLookup: %v", err)
			}
			if lookup.Attributes["instanceType"] != tt.wantInstance {
				t.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], tt.wantInstance)
			}
		})
	}
}

func TestEKSNodeGroupHandler_CalculateCost(t *testing.T) {
	h := &EKSNodeGroupHandler{}
	price := &pricing.Price{OnDemandUSD: 0.10}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantHourly float64
	}{
		{"default 1 node", map[string]any{}, 0.10},
		{"3 nodes", map[string]any{
			"scaling_config": []any{map[string]any{"desired_size": 3}},
		}, 0.30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hourly, _ := h.CalculateCost(price, tt.attrs)
			if diff := hourly - tt.wantHourly; diff < -0.001 || diff > 0.001 {
				t.Errorf("hourly = %v, want %v", hourly, tt.wantHourly)
			}
		})
	}
}
