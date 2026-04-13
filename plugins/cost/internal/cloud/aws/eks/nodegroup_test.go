package eks

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/contracttest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestNodeGroupHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	price := &pricing.Price{OnDemandUSD: 0.10}
	def := resourcespec.MustCompileTyped(NodeGroupSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

	tests := []struct {
		name       string
		attrs      map[string]any
		wantHourly float64
	}{
		{"default 1 node", map[string]any{}, 0.10},
		{"3 nodes", map[string]any{
			"scaling_config": []any{map[string]any{"desired_size": 3}},
		}, 0.30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hourly, _, ok := def.CalculateStandardCost(price, nil, "", tt.attrs)
			if !ok {
				t.Fatal("CalculateStandardCost() ok = false, want true")
			}
			if diff := hourly - tt.wantHourly; diff < -0.001 || diff > 0.001 {
				t.Errorf("hourly = %v, want %v", hourly, tt.wantHourly)
			}
		})
	}
}

func TestNodeGroupHandler_Contract(t *testing.T) {
	t.Parallel()

	category := resourcedef.CostCategoryStandard
	contracttest.RunContractSuite(t, resourcespec.MustCompileTyped(NodeGroupSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))), contracttest.ContractSuite{
		Category: &category,
		LookupCases: []contracttest.LookupCase{
			{
				Name:   "with instance_types",
				Region: "us-east-1",
				Attrs:  map[string]any{"instance_types": []any{"m5.large"}},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["instanceType"] != "m5.large" {
						tb.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], "m5.large")
					}
				},
			},
			{
				Name:   "with typed instance_types slice",
				Region: "us-east-1",
				Attrs:  map[string]any{"instance_types": []string{"c6i.large"}},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["instanceType"] != "c6i.large" {
						tb.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], "c6i.large")
					}
				},
			},
			{
				Name:   "default",
				Region: "us-east-1",
				Attrs:  map[string]any{},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.Attributes["instanceType"] != "t3.medium" {
						tb.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], "t3.medium")
					}
				},
			},
		},
		DescribeCases: []contracttest.DescribeCase{
			{
				Name:       "nil attrs",
				Attrs:      nil,
				WantAbsent: []string{"instance_type", "desired_size"},
			},
			{
				Name:       "empty attrs",
				Attrs:      map[string]any{},
				WantAbsent: []string{"instance_type", "desired_size"},
			},
			{
				Name: "instance_types and scaling_config",
				Attrs: map[string]any{
					"instance_types": []any{"m5.large"},
					"scaling_config": []any{map[string]any{"desired_size": float64(3)}},
				},
				WantKeys: map[string]string{
					"instance_type": "m5.large",
					"desired_size":  "3",
				},
			},
			{
				Name: "instance_types only",
				Attrs: map[string]any{
					"instance_types": []any{"t3.small"},
				},
				WantKeys:   map[string]string{"instance_type": "t3.small"},
				WantAbsent: []string{"desired_size"},
			},
			{
				Name: "scaling_config only",
				Attrs: map[string]any{
					"scaling_config": []any{map[string]any{"desired_size": float64(5)}},
				},
				WantKeys:   map[string]string{"desired_size": "5"},
				WantAbsent: []string{"instance_type"},
			},
		},
	})
}
