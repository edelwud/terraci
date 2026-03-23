package eks

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestNodeGroupHandler_Category(t *testing.T) {
	h := &NodeGroupHandler{}
	if h.Category() != aws.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestNodeGroupHandler_ServiceCode(t *testing.T) {
	h := &NodeGroupHandler{}
	if h.ServiceCode() != pricing.ServiceEC2 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceEC2)
	}
}

func TestNodeGroupHandler_BuildLookup(t *testing.T) {
	h := &NodeGroupHandler{}

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

func TestNodeGroupHandler_CalculateCost(t *testing.T) {
	h := &NodeGroupHandler{}
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
			hourly, _ := h.CalculateCost(price, nil, "", tt.attrs)
			if diff := hourly - tt.wantHourly; diff < -0.001 || diff > 0.001 {
				t.Errorf("hourly = %v, want %v", hourly, tt.wantHourly)
			}
		})
	}
}

func TestNodeGroupHandler_Describe(t *testing.T) {
	h := &NodeGroupHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantKeys   map[string]string
		wantAbsent []string
	}{
		{
			name:       "nil attrs",
			attrs:      nil,
			wantAbsent: []string{"instance_type", "desired_size"},
		},
		{
			name:       "empty attrs",
			attrs:      map[string]any{},
			wantAbsent: []string{"instance_type", "desired_size"},
		},
		{
			name: "instance_types and scaling_config",
			attrs: map[string]any{
				"instance_types": []any{"m5.large"},
				"scaling_config": []any{map[string]any{"desired_size": float64(3)}},
			},
			wantKeys: map[string]string{
				"instance_type": "m5.large",
				"desired_size":  "3",
			},
		},
		{
			name: "instance_types only",
			attrs: map[string]any{
				"instance_types": []any{"t3.small"},
			},
			wantKeys:   map[string]string{"instance_type": "t3.small"},
			wantAbsent: []string{"desired_size"},
		},
		{
			name: "scaling_config only",
			attrs: map[string]any{
				"scaling_config": []any{map[string]any{"desired_size": float64(5)}},
			},
			wantKeys:   map[string]string{"desired_size": "5"},
			wantAbsent: []string{"instance_type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Describe(nil, tt.attrs)

			for k, v := range tt.wantKeys {
				if result[k] != v {
					t.Errorf("Describe()[%q] = %q, want %q", k, result[k], v)
				}
			}
			for _, k := range tt.wantAbsent {
				if _, ok := result[k]; ok {
					t.Errorf("Describe() should not contain key %q", k)
				}
			}
		})
	}
}
