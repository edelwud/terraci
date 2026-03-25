package elasticache

import (
	"testing"

	aws "github.com/edelwud/terraci/plugins/cost/internal/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestClusterHandler_Category(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}
	if h.Category() != aws.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestClusterHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}

	attrs := map[string]any{
		"node_type":       "cache.t3.micro",
		"engine":          "redis",
		"num_cache_nodes": 3,
	}
	result := h.Describe(nil, attrs)

	if result["node_type"] != "cache.t3.micro" {
		t.Errorf("Describe()[node_type] = %q, want %q", result["node_type"], "cache.t3.micro")
	}
	if result["engine"] != "redis" {
		t.Errorf("Describe()[engine] = %q, want %q", result["engine"], "redis")
	}
	if result["nodes"] != "3" {
		t.Errorf("Describe()[nodes] = %q, want %q", result["nodes"], "3")
	}
}

func TestClusterHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}
	if h.ServiceCode() != pricing.ServiceElastiCache {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceElastiCache)
	}
}

func TestClusterHandler_BuildLookup(t *testing.T) {
	t.Parallel()

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
			if lookup.Attributes["cacheEngine"] != tt.wantEngine {
				t.Errorf("cacheEngine = %q, want %q", lookup.Attributes["cacheEngine"], tt.wantEngine)
			}
		})
	}
}

func TestClusterHandler_CalculateCost(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			hourly, _ := h.CalculateCost(price, nil, "", tt.attrs)
			if diff := hourly - tt.expectedHourly; diff < -epsilon || diff > epsilon {
				t.Errorf("hourly = %v, want %v", hourly, tt.expectedHourly)
			}
		})
	}
}

func TestClusterHandler_CalculateCost_BackupStorage(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}

	// Price with memory attribute from AWS API
	price := &pricing.Price{
		OnDemandUSD: 0.05,
		Attributes: map[string]string{
			"memory": "13.07 GiB",
		},
	}

	// 3 nodes, snapshot_retention_limit=5
	// Total backup = 13.07 * 3 * 5 = 196.05 GB
	// Free tier = 13.07 * 3 = 39.21 GB
	// Chargeable = 196.05 - 39.21 = 156.84 GB
	attrs := map[string]any{
		"node_type":                "cache.r5.large",
		"num_cache_nodes":          3,
		"snapshot_retention_limit": 5,
	}

	backupPrice := 0.090
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
		},
	}

	_, monthly := h.CalculateCost(price, index, "us-east-1", attrs)

	_, nodeMonthly := aws.ScaledHourlyCost(0.05, 3)
	chargeableGB := 13.07*3*5 - 13.07*3
	expectedMonthly := nodeMonthly + chargeableGB*backupPrice

	const epsilon = 0.01
	if diff := monthly - expectedMonthly; diff < -epsilon || diff > epsilon {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestClusterHandler_CalculateCost_DataTiering(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}

	// Price with SSD storage attribute from AWS API
	price := &pricing.Price{
		OnDemandUSD: 0.50,
		Attributes: map[string]string{
			"memory":  "26.32 GiB",
			"storage": "75 GiB NVMe SSD",
		},
	}

	attrs := map[string]any{
		"node_type":       "cache.r6gd.xlarge",
		"num_cache_nodes": 2,
	}

	dtPrice := 0.015
	index := &pricing.PriceIndex{
		Products: map[string]pricing.Price{
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

	_, nodeMonthly := aws.ScaledHourlyCost(0.50, 2)
	ssdCost := 75.0 * 2 * dtPrice
	expectedMonthly := nodeMonthly + ssdCost

	const epsilon = 0.01
	if diff := monthly - expectedMonthly; diff < -epsilon || diff > epsilon {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestClusterHandler_CalculateCost_Fallback(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}

	// Price with storage/memory from AWS API, empty index → fallback pricing
	price := &pricing.Price{
		OnDemandUSD: 0.50,
		Attributes: map[string]string{
			"memory":  "26.32 GiB",
			"storage": "75 GiB NVMe SSD",
		},
	}

	attrs := map[string]any{
		"node_type":                "cache.r6gd.xlarge",
		"num_cache_nodes":          1,
		"snapshot_retention_limit": 3,
	}

	index := &pricing.PriceIndex{
		Products: map[string]pricing.Price{},
	}

	_, monthly := h.CalculateCost(price, index, "us-east-1", attrs)

	_, nodeMonthly := aws.ScaledHourlyCost(0.50, 1)
	ssdCost := 75.0 * 1 * FallbackDataTieringCostPerGBMonth
	chargeableGB := 26.32*1*3 - 26.32*1
	backupCost := chargeableGB * FallbackBackupStorageCostPerGBMonth
	expectedMonthly := nodeMonthly + ssdCost + backupCost

	const epsilon = 0.01
	if diff := monthly - expectedMonthly; diff < -epsilon || diff > epsilon {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestClusterHandler_NoBackupCostWithRetention1(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}

	price := &pricing.Price{
		OnDemandUSD: 0.05,
		Attributes: map[string]string{
			"memory": "13.07 GiB",
		},
	}

	// snapshot_retention_limit=1 means 1 snapshot, which equals free tier
	attrs := map[string]any{
		"node_type":                "cache.r5.large",
		"num_cache_nodes":          1,
		"snapshot_retention_limit": 1,
	}

	_, monthly := h.CalculateCost(price, nil, "", attrs)

	_, nodeMonthly := aws.ScaledHourlyCost(0.05, 1)
	// chargeableGB = 13.07*1*1 - 13.07*1 = 0, so no backup cost
	if monthly != nodeMonthly {
		t.Errorf("monthly = %v, want %v (no backup cost for retention=1)", monthly, nodeMonthly)
	}
}

func TestClusterHandler_NoStorageCostWithoutSSD(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}

	// Non-SSD node — storage attribute is "None"
	price := &pricing.Price{
		OnDemandUSD: 0.05,
		Attributes: map[string]string{
			"memory":  "13.07 GiB",
			"storage": "None",
		},
	}

	attrs := map[string]any{
		"node_type":       "cache.r5.large",
		"num_cache_nodes": 1,
	}

	_, monthly := h.CalculateCost(price, nil, "", attrs)

	_, nodeMonthly := aws.ScaledHourlyCost(0.05, 1)
	if monthly != nodeMonthly {
		t.Errorf("monthly = %v, want %v (no SSD cost for non-SSD node)", monthly, nodeMonthly)
	}
}
