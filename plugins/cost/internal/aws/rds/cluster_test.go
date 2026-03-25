package rds

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

func TestClusterHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}
	if h.ServiceCode() != pricing.ServiceRDS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceRDS)
	}
}

func TestClusterHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup == nil {
		t.Fatal("expected non-nil lookup for aurora storage")
	}
	if lookup.ServiceCode != pricing.ServiceRDS {
		t.Errorf("ServiceCode = %q, want %q", lookup.ServiceCode, pricing.ServiceRDS)
	}
	if lookup.ProductFamily != "Database Storage" {
		t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Database Storage")
	}
	if lookup.Attributes["volumeType"] != "Aurora:StorageUsage" {
		t.Errorf("volumeType = %q, want %q", lookup.Attributes["volumeType"], "Aurora:StorageUsage")
	}
}

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

			expectedHourly := tt.wantMonthly / aws.HoursPerMonth
			if hourly != expectedHourly {
				t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
			}
		})
	}
}

func TestClusterHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &ClusterHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantKeys   map[string]string
		wantAbsent []string
	}{
		{
			name:       "nil attrs",
			attrs:      nil,
			wantAbsent: []string{"engine", "storage_gb"},
		},
		{
			name: "engine only",
			attrs: map[string]any{
				"engine": "aurora-postgresql",
			},
			wantKeys: map[string]string{"engine": "aurora-postgresql"},
		},
		{
			name: "engine and storage",
			attrs: map[string]any{
				"engine":            "aurora-mysql",
				"allocated_storage": float64(100),
			},
			wantKeys: map[string]string{
				"engine":     "aurora-mysql",
				"storage_gb": "100",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
