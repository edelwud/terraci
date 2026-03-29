package rds

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestClusterInstanceHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	price := &pricing.Price{OnDemandUSD: 0.29}
	h := &ClusterInstanceHandler{}
	hourly, monthly := h.CalculateCost(price, nil, "", nil)

	if hourly != 0.29 {
		t.Errorf("hourly = %v, want 0.29", hourly)
	}
	if monthly != 0.29*handler.HoursPerMonth {
		t.Errorf("monthly = %v, want %v", monthly, 0.29*handler.HoursPerMonth)
	}
}

func TestClusterInstanceHandler_Contract(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryStandard
	handlertest.RunContractSuite(t, &ClusterInstanceHandler{}, handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
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
				Name:    "missing instance_class",
				Region:  "us-east-1",
				Attrs:   map[string]any{},
				WantErr: true,
			},
		},
		DescribeCases: []handlertest.DescribeCase{
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
