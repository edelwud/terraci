package rds

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestInstanceHandler_Contract(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryStandard
	handlertest.RunContractSuite(t, resourcespec.MustHandler(InstanceSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))), handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
			{
				Name:   "mysql single-az",
				Region: "us-east-1",
				Attrs: map[string]any{
					"instance_class": "db.t3.micro",
					"engine":         "mysql",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["instanceType"] != "db.t3.micro" {
						tb.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], "db.t3.micro")
					}
					if lookup.Attributes["databaseEngine"] != "MySQL" {
						tb.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], "MySQL")
					}
					if lookup.Attributes["deploymentOption"] != "Single-AZ" {
						tb.Errorf("deploymentOption = %q, want %q", lookup.Attributes["deploymentOption"], "Single-AZ")
					}
				},
			},
			{
				Name:   "postgres multi-az",
				Region: "eu-central-1",
				Attrs: map[string]any{
					"instance_class": "db.m5.large",
					"engine":         "postgres",
					"multi_az":       true,
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["instanceType"] != "db.m5.large" {
						tb.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], "db.m5.large")
					}
					if lookup.Attributes["databaseEngine"] != "PostgreSQL" {
						tb.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], "PostgreSQL")
					}
					if lookup.Attributes["deploymentOption"] != "Multi-AZ" {
						tb.Errorf("deploymentOption = %q, want %q", lookup.Attributes["deploymentOption"], "Multi-AZ")
					}
				},
			},
			{
				Name:   "aurora-mysql",
				Region: "us-west-2",
				Attrs: map[string]any{
					"instance_class": "db.r5.large",
					"engine":         "aurora-mysql",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["instanceType"] != "db.r5.large" {
						tb.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], "db.r5.large")
					}
					if lookup.Attributes["databaseEngine"] != "Aurora MySQL" {
						tb.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], "Aurora MySQL")
					}
					if lookup.Attributes["deploymentOption"] != "Single-AZ" {
						tb.Errorf("deploymentOption = %q, want %q", lookup.Attributes["deploymentOption"], "Single-AZ")
					}
				},
			},
			{
				Name:    "missing instance_class",
				Region:  "us-east-1",
				Attrs:   map[string]any{},
				WantErr: true,
			},
		},
		DescribeCases: []handlertest.DescribeCase{
			{
				Name: "instance description",
				Attrs: map[string]any{
					"instance_class":    "db.t3.micro",
					"engine":            "postgres",
					"multi_az":          true,
					"allocated_storage": float64(100),
				},
				WantKeys: map[string]string{
					"instance_class": "db.t3.micro",
					"engine":         "postgres",
					"multi_az":       "true",
					"storage_gb":     "100",
				},
			},
		},
	})
}

func TestInstanceHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(InstanceSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.StandardCostHandler)
	if !ok {
		t.Fatal("handler should implement StandardCostHandler")
	}

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

			result := mapRDSEngine(tt.engine)
			if result != tt.expected {
				t.Errorf("mapRDSEngine(%q) = %q, want %q", tt.engine, result, tt.expected)
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

			result := getStorageCostPerGB(tt.storageType)
			if result != tt.expected {
				t.Errorf("getStorageCostPerGB(%q) = %v, want %v", tt.storageType, result, tt.expected)
			}
		})
	}
}
