package rds

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestClusterHandler_CalculateCost(t *testing.T) {
	t.Parallel()
	h := &ClusterHandler{}

	tests := []struct {
		name        string
		attrs       map[string]any
		wantMonthly float64
	}{
		{
			name:        "default 10GB minimum",
			attrs:       nil,
			wantMonthly: 10 * AuroraStorageCostPerGB,
		},
		{
			name:        "zero allocated_storage uses default",
			attrs:       map[string]any{},
			wantMonthly: 10 * AuroraStorageCostPerGB,
		},
		{
			name: "custom allocated_storage 50GB",
			attrs: map[string]any{
				"allocated_storage": float64(50),
			},
			wantMonthly: 50 * AuroraStorageCostPerGB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hourly, monthly := h.CalculateCost(nil, nil, "", tt.attrs)

			if monthly != tt.wantMonthly {
				t.Errorf("monthly = %v, want %v", monthly, tt.wantMonthly)
			}

			expectedHourly := tt.wantMonthly / handler.HoursPerMonth
			if hourly != expectedHourly {
				t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
			}
		})
	}
}

func TestClusterHandler_Contract(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryStandard
	handlertest.RunContractSuite(t, &ClusterHandler{}, handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
			{
				Name:   "default lookup",
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
		},
		DescribeCases: []handlertest.DescribeCase{
			{
				Name:       "nil attrs",
				Attrs:      nil,
				WantAbsent: []string{"engine", "storage_gb"},
			},
			{
				Name: "engine only",
				Attrs: map[string]any{
					"engine": "aurora-postgresql",
				},
				WantKeys: map[string]string{"engine": "aurora-postgresql"},
			},
			{
				Name: "engine and storage",
				Attrs: map[string]any{
					"engine":            "aurora-mysql",
					"allocated_storage": float64(100),
				},
				WantKeys: map[string]string{
					"engine":     "aurora-mysql",
					"storage_gb": "100",
				},
			},
		},
	})
}
