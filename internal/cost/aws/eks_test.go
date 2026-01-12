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

func TestEKSClusterHandler_BuildLookup(t *testing.T) {
	h := &EKSClusterHandler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}

	if lookup.ProductFamily != "Compute" {
		t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Compute")
	}

	expectedUsageType := "us-east-1-AmazonEKS-Hours:perCluster"
	if lookup.Attributes["usagetype"] != expectedUsageType {
		t.Errorf("usagetype = %q, want %q", lookup.Attributes["usagetype"], expectedUsageType)
	}
}

func TestEKSClusterHandler_CalculateCost(t *testing.T) {
	h := &EKSClusterHandler{}

	// With price
	price := &pricing.Price{OnDemandUSD: 0.10}
	hourly, monthly := h.CalculateCost(price, nil)
	if hourly != 0.10 {
		t.Errorf("hourly = %v, want %v", hourly, 0.10)
	}
	if monthly != 0.10*730 {
		t.Errorf("monthly = %v, want %v", monthly, 0.10*730)
	}

	// Fallback
	hourly, _ = h.CalculateCost(&pricing.Price{OnDemandUSD: 0}, nil)
	if hourly != 0.10 {
		t.Errorf("fallback hourly = %v, want %v", hourly, 0.10)
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
		attrs        map[string]interface{}
		wantInstance string
	}{
		{
			name: "with instance_types",
			attrs: map[string]interface{}{
				"instance_types": []interface{}{"m5.large"},
			},
			wantInstance: "m5.large",
		},
		{
			name:         "default",
			attrs:        map[string]interface{}{},
			wantInstance: "t3.medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup, err := h.BuildLookup("us-east-1", tt.attrs)
			if err != nil {
				t.Fatalf("BuildLookup returned error: %v", err)
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
		name           string
		attrs          map[string]interface{}
		expectedHourly float64
	}{
		{
			name:           "default 1 node",
			attrs:          map[string]interface{}{},
			expectedHourly: 0.10,
		},
		{
			name: "3 nodes",
			attrs: map[string]interface{}{
				"scaling_config": []interface{}{
					map[string]interface{}{
						"desired_size": 3,
					},
				},
			},
			expectedHourly: 0.30,
		},
	}

	const epsilon = 0.0001
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hourly, _ := h.CalculateCost(price, tt.attrs)
			if diff := hourly - tt.expectedHourly; diff < -epsilon || diff > epsilon {
				t.Errorf("hourly = %v, want %v", hourly, tt.expectedHourly)
			}
		})
	}
}
