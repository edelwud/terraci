package rds

import (
	"testing"

	aws "github.com/edelwud/terraci/plugins/cost/internal/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestClusterInstanceHandler_Category(t *testing.T) {
	t.Parallel()

	h := &ClusterInstanceHandler{}
	if h.Category() != aws.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestClusterInstanceHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &ClusterInstanceHandler{}
	if h.ServiceCode() != pricing.ServiceRDS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceRDS)
	}
}

func TestClusterInstanceHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &ClusterInstanceHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantErr    bool
		wantClass  string
		wantEngine string
	}{
		{
			name: "aurora-mysql instance",
			attrs: map[string]any{
				"instance_class": "db.r5.large",
				"engine":         "aurora-mysql",
			},
			wantClass:  "db.r5.large",
			wantEngine: "Aurora MySQL",
		},
		{
			name: "aurora-postgresql instance",
			attrs: map[string]any{
				"instance_class": "db.r5.xlarge",
				"engine":         "aurora-postgresql",
			},
			wantClass:  "db.r5.xlarge",
			wantEngine: "Aurora PostgreSQL",
		},
		{
			name:    "missing instance_class",
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

			if lookup.Attributes["instanceType"] != tt.wantClass {
				t.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], tt.wantClass)
			}
			if lookup.Attributes["databaseEngine"] != tt.wantEngine {
				t.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], tt.wantEngine)
			}
		})
	}
}

func TestClusterInstanceHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &ClusterInstanceHandler{}

	price := &pricing.Price{OnDemandUSD: 0.29}
	hourly, monthly := h.CalculateCost(price, nil, "", nil)

	if hourly != 0.29 {
		t.Errorf("hourly = %v, want 0.29", hourly)
	}
	if monthly != 0.29*aws.HoursPerMonth {
		t.Errorf("monthly = %v, want %v", monthly, 0.29*aws.HoursPerMonth)
	}
}

func TestClusterInstanceHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &ClusterInstanceHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantKeys   map[string]string
		wantAbsent []string
	}{
		{
			name:       "nil attrs",
			attrs:      nil,
			wantAbsent: []string{"instance_class", "engine"},
		},
		{
			name: "instance_class and engine",
			attrs: map[string]any{
				"instance_class": "db.r5.large",
				"engine":         "aurora-postgresql",
			},
			wantKeys: map[string]string{
				"instance_class": "db.r5.large",
				"engine":         "aurora-postgresql",
			},
		},
		{
			name: "instance_class only",
			attrs: map[string]any{
				"instance_class": "db.r6g.xlarge",
			},
			wantKeys:   map[string]string{"instance_class": "db.r6g.xlarge"},
			wantAbsent: []string{"engine"},
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
