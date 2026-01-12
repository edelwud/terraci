package aws

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestElastiCacheClusterHandler_ServiceCode(t *testing.T) {
	h := &ElastiCacheClusterHandler{}
	if h.ServiceCode() != pricing.ServiceElastiCache {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceElastiCache)
	}
}

func TestElastiCacheClusterHandler_BuildLookup(t *testing.T) {
	h := &ElastiCacheClusterHandler{}

	tests := []struct {
		name       string
		attrs      map[string]interface{}
		wantErr    bool
		wantType   string
		wantEngine string
	}{
		{
			name: "redis cluster",
			attrs: map[string]interface{}{
				"node_type": "cache.t3.micro",
				"engine":    "redis",
			},
			wantType:   "cache.t3.micro",
			wantEngine: "Redis",
		},
		{
			name: "memcached cluster",
			attrs: map[string]interface{}{
				"node_type": "cache.m5.large",
				"engine":    "memcached",
			},
			wantType:   "cache.m5.large",
			wantEngine: "Memcached",
		},
		{
			name: "default engine",
			attrs: map[string]interface{}{
				"node_type": "cache.t3.micro",
			},
			wantType:   "cache.t3.micro",
			wantEngine: "Redis",
		},
		{
			name:    "missing node_type",
			attrs:   map[string]interface{}{},
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

func TestElastiCacheClusterHandler_CalculateCost(t *testing.T) {
	h := &ElastiCacheClusterHandler{}

	price := &pricing.Price{OnDemandUSD: 0.05}

	tests := []struct {
		name           string
		attrs          map[string]interface{}
		expectedHourly float64
	}{
		{
			name:           "single node",
			attrs:          map[string]interface{}{},
			expectedHourly: 0.05,
		},
		{
			name: "3 nodes",
			attrs: map[string]interface{}{
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

func TestElastiCacheReplicationGroupHandler_ServiceCode(t *testing.T) {
	h := &ElastiCacheReplicationGroupHandler{}
	if h.ServiceCode() != pricing.ServiceElastiCache {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceElastiCache)
	}
}

func TestElastiCacheReplicationGroupHandler_BuildLookup(t *testing.T) {
	h := &ElastiCacheReplicationGroupHandler{}

	tests := []struct {
		name     string
		attrs    map[string]interface{}
		wantErr  bool
		wantType string
	}{
		{
			name: "replication group",
			attrs: map[string]interface{}{
				"node_type": "cache.r5.large",
			},
			wantType: "cache.r5.large",
		},
		{
			name:    "missing node_type",
			attrs:   map[string]interface{}{},
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

func TestElastiCacheReplicationGroupHandler_CalculateCost(t *testing.T) {
	h := &ElastiCacheReplicationGroupHandler{}

	price := &pricing.Price{OnDemandUSD: 0.10}

	tests := []struct {
		name           string
		attrs          map[string]interface{}
		expectedNodes  int
		expectedHourly float64
	}{
		{
			name:           "default single node",
			attrs:          map[string]interface{}{},
			expectedNodes:  1,
			expectedHourly: 0.10,
		},
		{
			name: "2 shards with 2 replicas each",
			attrs: map[string]interface{}{
				"num_node_groups":         2,
				"replicas_per_node_group": 2,
			},
			expectedNodes:  6, // 2 shards * (1 primary + 2 replicas)
			expectedHourly: 0.60,
		},
		{
			name: "legacy number_cache_clusters",
			attrs: map[string]interface{}{
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
