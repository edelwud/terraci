package elasticache

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestReplicationGroupHandler_ServiceCode(t *testing.T) {
	h := &ReplicationGroupHandler{}
	if h.ServiceCode() != pricing.ServiceElastiCache {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceElastiCache)
	}
}

func TestReplicationGroupHandler_BuildLookup(t *testing.T) {
	h := &ReplicationGroupHandler{}

	tests := []struct {
		name     string
		attrs    map[string]any
		wantErr  bool
		wantType string
	}{
		{
			name: "replication group",
			attrs: map[string]any{
				"node_type": "cache.r5.large",
			},
			wantType: "cache.r5.large",
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
		})
	}
}

func TestReplicationGroupHandler_CalculateCost(t *testing.T) {
	h := &ReplicationGroupHandler{}

	price := &pricing.Price{OnDemandUSD: 0.10}

	tests := []struct {
		name           string
		attrs          map[string]any
		expectedNodes  int
		expectedHourly float64
	}{
		{
			name:           "default single node",
			attrs:          map[string]any{},
			expectedNodes:  1,
			expectedHourly: 0.10,
		},
		{
			name: "2 shards with 2 replicas each",
			attrs: map[string]any{
				"num_node_groups":         2,
				"replicas_per_node_group": 2,
			},
			expectedNodes:  6, // 2 shards * (1 primary + 2 replicas)
			expectedHourly: 0.60,
		},
		{
			name: "legacy number_cache_clusters",
			attrs: map[string]any{
				"number_cache_clusters": 3,
			},
			expectedNodes:  3,
			expectedHourly: 0.30,
		},
	}

	const epsilon = 0.0001
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hourly, _ := h.CalculateCost(price, tt.attrs)
			if diff := hourly - tt.expectedHourly; diff < -epsilon || diff > epsilon {
				t.Errorf("hourly = %v, want %v (expected %d nodes)", hourly, tt.expectedHourly, tt.expectedNodes)
			}
		})
	}
}
