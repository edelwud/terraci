package aws

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestRDSInstanceHandler_ServiceCode(t *testing.T) {
	h := &RDSInstanceHandler{}
	if h.ServiceCode() != pricing.ServiceRDS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceRDS)
	}
}

func TestRDSInstanceHandler_BuildLookup(t *testing.T) {
	h := &RDSInstanceHandler{}

	tests := []struct {
		name           string
		region         string
		attrs          map[string]interface{}
		wantErr        bool
		wantClass      string
		wantEngine     string
		wantDeployment string
	}{
		{
			name:   "mysql single-az",
			region: "us-east-1",
			attrs: map[string]interface{}{
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
			attrs: map[string]interface{}{
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
			attrs: map[string]interface{}{
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
			attrs:   map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestRDSInstanceHandler_CalculateCost(t *testing.T) {
	h := &RDSInstanceHandler{}

	price := &pricing.Price{
		OnDemandUSD: 0.10, // $0.10/hour
	}

	tests := []struct {
		name            string
		attrs           map[string]interface{}
		expectedMonthly float64
	}{
		{
			name:            "compute only",
			attrs:           map[string]interface{}{},
			expectedMonthly: 0.10 * 730,
		},
		{
			name: "with gp2 storage",
			attrs: map[string]interface{}{
				"storage_type":      "gp2",
				"allocated_storage": float64(100),
			},
			expectedMonthly: 0.10*730 + 100*0.115, // compute + storage
		},
		{
			name: "with io1 storage and iops",
			attrs: map[string]interface{}{
				"storage_type":      "io1",
				"allocated_storage": float64(100),
				"iops":              float64(1000),
			},
			expectedMonthly: 0.10*730 + 100*0.125 + 1000*0.10, // compute + storage + iops
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, monthly := h.CalculateCost(price, tt.attrs)

			if monthly != tt.expectedMonthly {
				t.Errorf("monthly = %v, want %v", monthly, tt.expectedMonthly)
			}
		})
	}
}

func TestRDSClusterHandler_ServiceCode(t *testing.T) {
	h := &RDSClusterHandler{}
	if h.ServiceCode() != pricing.ServiceRDS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceRDS)
	}
}

func TestRDSClusterInstanceHandler_BuildLookup(t *testing.T) {
	h := &RDSClusterInstanceHandler{}

	tests := []struct {
		name       string
		attrs      map[string]interface{}
		wantErr    bool
		wantClass  string
		wantEngine string
	}{
		{
			name: "aurora-mysql instance",
			attrs: map[string]interface{}{
				"instance_class": "db.r5.large",
				"engine":         "aurora-mysql",
			},
			wantClass:  "db.r5.large",
			wantEngine: "Aurora MySQL",
		},
		{
			name: "aurora-postgresql instance",
			attrs: map[string]interface{}{
				"instance_class": "db.r5.xlarge",
				"engine":         "aurora-postgresql",
			},
			wantClass:  "db.r5.xlarge",
			wantEngine: "Aurora PostgreSQL",
		},
		{
			name:    "missing instance_class",
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

			if lookup.Attributes["instanceType"] != tt.wantClass {
				t.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], tt.wantClass)
			}
			if lookup.Attributes["databaseEngine"] != tt.wantEngine {
				t.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], tt.wantEngine)
			}
		})
	}
}

func TestMapRDSEngine(t *testing.T) {
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
			result := mapRDSEngine(tt.engine)
			if result != tt.expected {
				t.Errorf("mapRDSEngine(%q) = %q, want %q", tt.engine, result, tt.expected)
			}
		})
	}
}

func TestGetStorageCostPerGB(t *testing.T) {
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
			result := getStorageCostPerGB(tt.storageType)
			if result != tt.expected {
				t.Errorf("getStorageCostPerGB(%q) = %v, want %v", tt.storageType, result, tt.expected)
			}
		})
	}
}
