package elasticache

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestClusterHandler_ServiceCode(t *testing.T) {
	h := &ClusterHandler{}
	if h.ServiceCode() != pricing.ServiceElastiCache {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceElastiCache)
	}
}

func TestClusterHandler_BuildLookup(t *testing.T) {
	h := &ClusterHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantErr    bool
		wantType   string
		wantEngine string
	}{
		{
			name: "redis cluster",
			attrs: map[string]any{
				"node_type": "cache.t3.micro",
				"engine":    "redis",
			},
			wantType:   "cache.t3.micro",
			wantEngine: "Redis",
		},
		{
			name: "memcached cluster",
			attrs: map[string]any{
				"node_type": "cache.m5.large",
				"engine":    "memcached",
			},
			wantType:   "cache.m5.large",
			wantEngine: "Memcached",
		},
		{
			name: "default engine",
			attrs: map[string]any{
				"node_type": "cache.t3.micro",
			},
			wantType:   "cache.t3.micro",
			wantEngine: "Redis",
		},
		{
			name:    "missing node_type",
			attrs:   map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup, err := h.BuildLookup("us-east-1", tt.attrs)

			if tt.wantErr {
				if err == nil {
					t.Error("BuildLookup should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("BuildLookup returned error: %v", err)
			}

			if lookup.Attributes["instanceType"] != tt.wantType {
				t.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], tt.wantType)
			}
			if lookup.Attributes["cacheEngine"] != tt.wantEngine {
				t.Errorf("cacheEngine = %q, want %q", lookup.Attributes["cacheEngine"], tt.wantEngine)
			}
		})
	}
}

func TestClusterHandler_CalculateCost(t *testing.T) {
	h := &ClusterHandler{}

	price := &pricing.Price{OnDemandUSD: 0.05}

	tests := []struct {
		name           string
		attrs          map[string]any
		expectedHourly float64
	}{
		{
			name:           "single node",
			attrs:          map[string]any{},
			expectedHourly: 0.05,
		},
		{
			name: "3 nodes",
			attrs: map[string]any{
				"num_cache_nodes": 3,
			},
			expectedHourly: 0.15,
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
