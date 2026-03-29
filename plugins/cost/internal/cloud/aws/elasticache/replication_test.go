package elasticache

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestReplicationGroupHandler_Contract(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryStandard
	handlertest.RunContractSuite(t, &ReplicationGroupHandler{}, handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
			{
				Name:   "replication group",
				Region: "us-east-1",
				Attrs: map[string]any{
					"node_type": "cache.r5.large",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["instanceType"] != "cache.r5.large" {
						tb.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], "cache.r5.large")
					}
				},
			},
			{
				Name:    "missing node_type",
				Region:  "us-east-1",
				Attrs:   map[string]any{},
				WantErr: true,
			},
		},
		DescribeCases: []handlertest.DescribeCase{
			{
				Name: "replication group description",
				Attrs: map[string]any{
					"node_type":             "cache.r5.large",
					"num_node_groups":       2,
					"number_cache_clusters": 4,
				},
				WantKeys: map[string]string{
					"node_type":   "cache.r5.large",
					"node_groups": "2",
				},
			},
		},
	})
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
	_, nodeMonthly := handler.ScaledHourlyCost(0.50, totalNodes)
	ssdCost := 150.0 * float64(totalNodes) * dtPrice
	chargeableGB := 52.82*float64(totalNodes)*3 - 52.82*float64(totalNodes)
	backupCost := chargeableGB * backupPrice
	expectedMonthly := nodeMonthly + ssdCost + backupCost

	const epsilon = 0.01
	if diff := monthly - expectedMonthly; diff < -epsilon || diff > epsilon {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}
