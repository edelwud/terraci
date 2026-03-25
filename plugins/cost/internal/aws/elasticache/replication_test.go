package elasticache

import (
	"testing"

	aws "github.com/edelwud/terraci/plugins/cost/internal/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestReplicationGroupHandler_Category(t *testing.T) {
	t.Parallel()

	h := &ReplicationGroupHandler{}
	if h.Category() != aws.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestReplicationGroupHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &ReplicationGroupHandler{}

	attrs := map[string]any{
		"node_type":             "cache.r5.large",
		"num_node_groups":       2,
		"number_cache_clusters": 4,
	}
	result := h.Describe(nil, attrs)

	if result["node_type"] != "cache.r5.large" {
		t.Errorf("Describe()[node_type] = %q, want %q", result["node_type"], "cache.r5.large")
	}
	if result["node_groups"] != "2" {
		t.Errorf("Describe()[node_groups] = %q, want %q", result["node_groups"], "2")
	}
}

func TestReplicationGroupHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &ReplicationGroupHandler{}
	if h.ServiceCode() != pricing.ServiceElastiCache {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceElastiCache)
	}
}

func TestReplicationGroupHandler_BuildLookup(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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
	t.Parallel()

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
			t.Parallel()

			hourly, _ := h.CalculateCost(price, nil, "", tt.attrs)
			if diff := hourly - tt.expectedHourly; diff < -epsilon || diff > epsilon {
				t.Errorf("hourly = %v, want %v (expected %d nodes)", hourly, tt.expectedHourly, tt.expectedNodes)
			}
		})
	}
}

func TestReplicationGroupHandler_CalculateCost_BackupAndDataTiering(t *testing.T) {
	t.Parallel()

	h := &ReplicationGroupHandler{}

	// Price with memory and SSD from AWS API
	price := &pricing.Price{
		OnDemandUSD: 0.50,
		Attributes: map[string]string{
			"memory":  "52.82 GiB",
			"storage": "150 GiB NVMe SSD",
		},
	}

	// 2 shards * (1 primary + 1 replica) = 4 nodes
	// snapshot_retention_limit = 3
	attrs := map[string]any{
		"node_type":                "cache.r6gd.2xlarge",
		"num_node_groups":          2,
		"replicas_per_node_group":  1,
		"snapshot_retention_limit": 3,
	}

	backupPrice := 0.090
	dtPrice := 0.015
	index := &pricing.PriceIndex{
		Products: map[string]pricing.Price{
			"backup-sku": {
				ProductFamily: "Storage Snapshot",
				Attributes: map[string]string{
					"usagetype": "USE1-BackupUsage",
					"location":  "US East (N. Virginia)",
				},
				OnDemandUSD: backupPrice,
			},
			"dt-sku": {
				ProductFamily: "Cache Storage",
				Attributes: map[string]string{
					"usagetype": "USE1-DataTiering:StorageUsage",
					"location":  "US East (N. Virginia)",
				},
				OnDemandUSD: dtPrice,
			},
		},
	}

	_, monthly := h.CalculateCost(price, index, "us-east-1", attrs)

	totalNodes := 4
	_, nodeMonthly := aws.ScaledHourlyCost(0.50, totalNodes)
	ssdCost := 150.0 * float64(totalNodes) * dtPrice
	chargeableGB := 52.82*float64(totalNodes)*3 - 52.82*float64(totalNodes)
	backupCost := chargeableGB * backupPrice
	expectedMonthly := nodeMonthly + ssdCost + backupCost

	const epsilon = 0.01
	if diff := monthly - expectedMonthly; diff < -epsilon || diff > epsilon {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}
