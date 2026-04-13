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

func TestClusterInstanceHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(ClusterInstanceSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	t.Run("standard hourly", func(t *testing.T) {
		t.Parallel()
		p := &pricing.Price{OnDemandUSD: 0.29}
		hourly, monthly, ok := def.CalculateStandardCost(p, nil, "", nil)
		if !ok {
			t.Fatal("CalculateStandardCost returned ok=false")
		}
		if hourly != 0.29 {
			t.Errorf("hourly = %v, want 0.29", hourly)
		}
		if monthly != 0.29*costutil.HoursPerMonth {
			t.Errorf("monthly = %v, want %v", monthly, 0.29*costutil.HoursPerMonth)
		}
	})

	t.Run("nil price returns zero", func(t *testing.T) {
		t.Parallel()
		hourly, monthly, ok := def.CalculateStandardCost(nil, nil, "", nil)
		if !ok {
			t.Fatal("CalculateStandardCost returned ok=false")
		}
		if hourly != 0 || monthly != 0 {
			t.Errorf("expected (0, 0), got (%v, %v)", hourly, monthly)
		}
	})
}

func TestClusterInstanceHandler_Contract(t *testing.T) {
	t.Parallel()

	category := resourcedef.CostCategoryStandard
	def := resourcespec.MustCompileTyped(ClusterInstanceSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))
	contracttest.RunContractSuite(t, def, contracttest.ContractSuite{
		Category: &category,
		LookupCases: []contracttest.LookupCase{
			{
				Name:   "aurora-mysql instance",
				Region: "us-east-1",
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
				},
			},
			{
				Name:   "aurora-postgresql instance",
				Region: "us-east-1",
				Attrs: map[string]any{
					"instance_class": "db.r5.xlarge",
					"engine":         "aurora-postgresql",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["instanceType"] != "db.r5.xlarge" {
						tb.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], "db.r5.xlarge")
					}
					if lookup.Attributes["databaseEngine"] != "Aurora PostgreSQL" {
						tb.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], "Aurora PostgreSQL")
					}
				},
			},
			{
				Name:   "default engine (empty) resolves to aurora-mysql",
				Region: "us-east-1",
				Attrs: map[string]any{
					"instance_class": "db.r6g.large",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["databaseEngine"] != "Aurora MySQL" {
						tb.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], "Aurora MySQL")
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
		DescribeCases: []contracttest.DescribeCase{
			{
				Name:       "nil attrs",
				Attrs:      nil,
				WantAbsent: []string{"instance_class", "engine"},
			},
			{
				Name: "instance_class and engine",
				Attrs: map[string]any{
					"instance_class": "db.r5.large",
					"engine":         "aurora-postgresql",
				},
				WantKeys: map[string]string{
					"instance_class": "db.r5.large",
					"engine":         "aurora-postgresql",
				},
			},
			{
				Name: "instance_class only",
				Attrs: map[string]any{
					"instance_class": "db.r6g.xlarge",
				},
				WantKeys:   map[string]string{"instance_class": "db.r6g.xlarge"},
				WantAbsent: []string{"engine"},
			},
		},
	})
}
