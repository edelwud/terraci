package rds

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/contracttest"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestClusterHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	tests := []struct {
		name        string
		attrs       map[string]any
		wantMonthly float64
	}{
		// Aurora variants
		{
			name:        "default engine (aurora) 10GB minimum",
			attrs:       nil,
			wantMonthly: 10 * AuroraStorageCostPerGB,
		},
		{
			name:        "empty attrs uses aurora 10GB default",
			attrs:       map[string]any{},
			wantMonthly: 10 * AuroraStorageCostPerGB,
		},
		{
			name: "aurora-mysql 50GB",
			attrs: map[string]any{
				"engine":            "aurora-mysql",
				"allocated_storage": float64(50),
			},
			wantMonthly: 50 * AuroraStorageCostPerGB,
		},
		{
			name: "aurora-postgresql 100GB",
			attrs: map[string]any{
				"engine":            "aurora-postgresql",
				"allocated_storage": float64(100),
			},
			wantMonthly: 100 * AuroraStorageCostPerGB,
		},
		{
			name: "aurora-mysql io-optimized 100GB",
			attrs: map[string]any{
				"engine":            "aurora-mysql",
				"storage_type":      "aurora-iopt1",
				"allocated_storage": float64(100),
			},
			wantMonthly: 100 * AuroraIOOptStorageCostPerGB,
		},
		{
			name: "aurora-postgresql io-optimized 50GB",
			attrs: map[string]any{
				"engine":            "aurora-postgresql",
				"storage_type":      "aurora-iopt1",
				"allocated_storage": float64(50),
			},
			wantMonthly: 50 * AuroraIOOptStorageCostPerGB,
		},
		// Non-aurora: MySQL
		{
			name: "mysql gp3 (default storage type)",
			attrs: map[string]any{
				"engine":            "mysql",
				"allocated_storage": float64(100),
			},
			wantMonthly: 100 * StorageCostGP3,
		},
		{
			name: "mysql io1",
			attrs: map[string]any{
				"engine":            "mysql",
				"storage_type":      "io1",
				"allocated_storage": float64(100),
			},
			wantMonthly: 100 * StorageCostIO1,
		},
		// Non-aurora: PostgreSQL
		{
			name: "postgres gp3",
			attrs: map[string]any{
				"engine":            "postgres",
				"storage_type":      "gp3",
				"allocated_storage": float64(50),
			},
			wantMonthly: 50 * StorageCostGP3,
		},
		{
			name: "postgres io2",
			attrs: map[string]any{
				"engine":            "postgres",
				"storage_type":      "io2",
				"allocated_storage": float64(200),
			},
			wantMonthly: 200 * StorageCostIO2,
		},
		// Edge cases
		{
			name: "aurora with explicit aurora storage_type",
			attrs: map[string]any{
				"engine":            "aurora-mysql",
				"storage_type":      "aurora",
				"allocated_storage": float64(75),
			},
			wantMonthly: 75 * AuroraStorageCostPerGB,
		},
		{
			name: "non-aurora without db_cluster_instance_class",
			attrs: map[string]any{
				"engine":            "postgres",
				"allocated_storage": float64(100),
			},
			// No db_cluster_instance_class → MultiAZ=false → no databaseEngine/deploymentOption in lookup
			// Fallback still works: storage_type defaults to gp3
			wantMonthly: 100 * StorageCostGP3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hourly, monthly, ok := def.CalculateStandardCost(nil, nil, "", tt.attrs)
			if !ok {
				t.Fatal("CalculateStandardCost returned ok=false")
			}

			if monthly != tt.wantMonthly {
				t.Errorf("monthly = %v, want %v", monthly, tt.wantMonthly)
			}

			expectedHourly := tt.wantMonthly / costutil.HoursPerMonth
			if hourly != expectedHourly {
				t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
			}
		})
	}
}

func TestClusterHandler_Contract(t *testing.T) {
	t.Parallel()

	category := resourcedef.CostCategoryStandard
	def := resourcespec.MustCompileTyped(ClusterSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))
	contracttest.RunContractSuite(t, def, contracttest.ContractSuite{
		Category: &category,
		LookupCases: []contracttest.LookupCase{
			{
				Name:   "default aurora lookup",
				Region: "us-east-1",
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.ServiceID != awskit.MustService(awskit.ServiceKeyRDS) {
						tb.Errorf("ServiceID = %q, want %q", lookup.ServiceID, awskit.MustService(awskit.ServiceKeyRDS))
					}
					if lookup.ProductFamily != "Database Storage" {
						tb.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Database Storage")
					}
					if lookup.Attributes["volumeType"] != "Aurora:StorageUsage" {
						tb.Errorf("volumeType = %q, want %q", lookup.Attributes["volumeType"], "Aurora:StorageUsage")
					}
				},
			},
			{
				Name:   "postgres multi-az cluster lookup",
				Region: "us-east-1",
				Attrs: map[string]any{
					"engine":                    "postgres",
					"storage_type":              "gp3",
					"db_cluster_instance_class": "db.r6gd.xlarge",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["databaseEngine"] != "PostgreSQL" {
						tb.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], "PostgreSQL")
					}
					if lookup.Attributes["deploymentOption"] != deploymentMultiAZReadableStandbys {
						tb.Errorf("deploymentOption = %q, want %q", lookup.Attributes["deploymentOption"], deploymentMultiAZReadableStandbys)
					}
					if lookup.Attributes["volumeType"] != "General Purpose-GP3" {
						tb.Errorf("volumeType = %q, want %q", lookup.Attributes["volumeType"], "General Purpose-GP3")
					}
				},
			},
			{
				Name:   "aurora-iopt1 lookup",
				Region: "us-east-1",
				Attrs: map[string]any{
					"engine":       "aurora-mysql",
					"storage_type": "aurora-iopt1",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["volumeType"] != "Aurora:StorageIOUsage" {
						tb.Errorf("volumeType = %q, want %q", lookup.Attributes["volumeType"], "Aurora:StorageIOUsage")
					}
					if _, ok := lookup.Attributes["databaseEngine"]; ok {
						tb.Errorf("databaseEngine should not be present for aurora")
					}
				},
			},
			{
				Name:   "mysql multi-az cluster lookup",
				Region: "us-east-1",
				Attrs: map[string]any{
					"engine":                    "mysql",
					"storage_type":              "io1",
					"db_cluster_instance_class": "db.r6gd.xlarge",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["databaseEngine"] != "MySQL" {
						tb.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], "MySQL")
					}
					if lookup.Attributes["deploymentOption"] != deploymentMultiAZReadableStandbys {
						tb.Errorf("deploymentOption = %q, want %q", lookup.Attributes["deploymentOption"], deploymentMultiAZReadableStandbys)
					}
					if lookup.Attributes["volumeType"] != "Provisioned IOPS" {
						tb.Errorf("volumeType = %q, want %q", lookup.Attributes["volumeType"], "Provisioned IOPS")
					}
				},
			},
		},
		DescribeCases: []contracttest.DescribeCase{
			{
				Name:       "nil attrs (aurora default)",
				Attrs:      nil,
				WantKeys:   map[string]string{"storage_type": "aurora"},
				WantAbsent: []string{"engine", "storage_gb"},
			},
			{
				Name: "aurora engine only",
				Attrs: map[string]any{
					"engine": "aurora-postgresql",
				},
				WantKeys: map[string]string{
					"engine":       "aurora-postgresql",
					"storage_type": "aurora",
				},
			},
			{
				Name: "engine and storage",
				Attrs: map[string]any{
					"engine":            "aurora-mysql",
					"storage_type":      "io1",
					"allocated_storage": float64(100),
				},
				WantKeys: map[string]string{
					"engine":       "aurora-mysql",
					"storage_type": "io1",
					"storage_gb":   "100",
				},
			},
		},
	})
}
