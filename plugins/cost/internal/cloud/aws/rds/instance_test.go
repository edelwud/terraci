package rds

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestInstanceHandler_Category(t *testing.T) {
	t.Parallel()

	h := &InstanceHandler{}
	if h.Category() != handler.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestInstanceHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &InstanceHandler{}

	attrs := map[string]any{
		"instance_class":    "db.t3.micro",
		"engine":            "postgres",
		"multi_az":          true,
		"allocated_storage": float64(100),
	}
	result := h.Describe(nil, attrs)

	if result["instance_class"] != "db.t3.micro" {
		t.Errorf("Describe()[instance_class] = %q, want %q", result["instance_class"], "db.t3.micro")
	}
	if result["engine"] != "postgres" {
		t.Errorf("Describe()[engine] = %q, want %q", result["engine"], "postgres")
	}
	if result["multi_az"] != "true" {
		t.Errorf("Describe()[multi_az] = %q, want %q", result["multi_az"], "true")
	}
	if result["storage_gb"] != "100" {
		t.Errorf("Describe()[storage_gb] = %q, want %q", result["storage_gb"], "100")
	}
}

func TestInstanceHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &InstanceHandler{}
	if h.ServiceCode() != pricing.ServiceRDS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceRDS)
	}
}

func TestInstanceHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &InstanceHandler{}

	tests := []struct {
		name           string
		region         string
		attrs          map[string]any
		wantErr        bool
		wantClass      string
		wantEngine     string
		wantDeployment string
	}{
		{
			name:   "mysql single-az",
			region: "us-east-1",
			attrs: map[string]any{
				"instance_class": "db.t3.micro",
				"engine":         "mysql",
			},
			wantClass:      "db.t3.micro",
			wantEngine:     "MySQL",
			wantDeployment: "Single-AZ",
		},
		{
			name:   "postgres multi-az",
			region: "eu-central-1",
			attrs: map[string]any{
				"instance_class": "db.m5.large",
				"engine":         "postgres",
				"multi_az":       true,
			},
			wantClass:      "db.m5.large",
			wantEngine:     "PostgreSQL",
			wantDeployment: "Multi-AZ",
		},
		{
			name:   "aurora-mysql",
			region: "us-west-2",
			attrs: map[string]any{
				"instance_class": "db.r5.large",
				"engine":         "aurora-mysql",
			},
			wantClass:      "db.r5.large",
			wantEngine:     "Aurora MySQL",
			wantDeployment: "Single-AZ",
		},
		{
			name:    "missing instance_class",
			region:  "us-east-1",
			attrs:   map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lookup, err := h.BuildLookup(tt.region, tt.attrs)

			if tt.wantErr {
				if err == nil {
					t.Error("BuildLookup should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("BuildLookup returned error: %v", err)
			}

			if lookup.Attributes["instanceType"] != tt.wantClass {
				t.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], tt.wantClass)
			}
			if lookup.Attributes["databaseEngine"] != tt.wantEngine {
				t.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], tt.wantEngine)
			}
			if lookup.Attributes["deploymentOption"] != tt.wantDeployment {
				t.Errorf("deploymentOption = %q, want %q", lookup.Attributes["deploymentOption"], tt.wantDeployment)
			}
		})
	}
}

func TestInstanceHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &InstanceHandler{}

	price := &pricing.Price{
		OnDemandUSD: 0.10, // $0.10/hour
	}

	tests := []struct {
		name            string
		attrs           map[string]any
		expectedMonthly float64
	}{
		{
			name:            "compute only",
			attrs:           map[string]any{},
			expectedMonthly: 0.10 * handler.HoursPerMonth,
		},
		{
			name: "with gp2 storage",
			attrs: map[string]any{
				"storage_type":      "gp2",
				"allocated_storage": float64(100),
			},
			expectedMonthly: 0.10*handler.HoursPerMonth + 100*0.115, // compute + storage
		},
		{
			name: "with io1 storage and iops",
			attrs: map[string]any{
				"storage_type":      "io1",
				"allocated_storage": float64(100),
				"iops":              float64(1000),
			},
			expectedMonthly: 0.10*handler.HoursPerMonth + 100*0.125 + 1000*0.10, // compute + storage + iops
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, monthly := h.CalculateCost(price, nil, "", tt.attrs)

			if monthly != tt.expectedMonthly {
				t.Errorf("monthly = %v, want %v", monthly, tt.expectedMonthly)
			}
		})
	}
}

func TestMapRDSEngine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		engine   string
		expected string
	}{
		{"mysql", "MySQL"},
		{"postgres", "PostgreSQL"},
		{"postgresql", "PostgreSQL"},
		{"mariadb", "MariaDB"},
		{"aurora-mysql", "Aurora MySQL"},
		{"aurora-postgresql", "Aurora PostgreSQL"},
		{"aurora", "Aurora MySQL"},
		{"oracle-se2", "Oracle"},
		{"sqlserver-ex", "SQL Server"},
		{"unknown", "MySQL"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			t.Parallel()

			result := MapRDSEngine(tt.engine)
			if result != tt.expected {
				t.Errorf("MapRDSEngine(%q) = %q, want %q", tt.engine, result, tt.expected)
			}
		})
	}
}

func TestGetStorageCostPerGB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		storageType string
		expected    float64
	}{
		{"gp2", 0.115},
		{"gp3", 0.115},
		{"io1", 0.125},
		{"standard", 0.10},
		{"unknown", 0.115},
	}

	for _, tt := range tests {
		t.Run(tt.storageType, func(t *testing.T) {
			t.Parallel()

			result := GetStorageCostPerGB(tt.storageType)
			if result != tt.expected {
				t.Errorf("GetStorageCostPerGB(%q) = %v, want %v", tt.storageType, result, tt.expected)
			}
		})
	}
}
