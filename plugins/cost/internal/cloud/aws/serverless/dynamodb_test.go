package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestDynamoDBHandler_CalculateUsageCost(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(DynamoDBSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.UsageBasedCostHandler)
	if !ok {
		t.Fatal("handler should implement UsageBasedCostHandler")
	}

	tests := []struct {
		name          string
		attrs         map[string]any
		wantHourly    float64
		wantMonthly   float64
		expectNonZero bool
	}{
		{
			name: "pay per request",
			attrs: map[string]any{
				"billing_mode": "PAY_PER_REQUEST",
			},
			wantHourly:  0,
			wantMonthly: 0,
		},
		{
			name:          "provisioned with defaults",
			attrs:         map[string]any{},
			expectNonZero: true,
		},
		{
			name: "provisioned with custom capacity",
			attrs: map[string]any{
				"read_capacity":  float64(10),
				"write_capacity": float64(20),
			},
			expectNonZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := h.CalculateUsageCost("", tt.attrs)

			if tt.expectNonZero {
				if got.HourlyCost == 0 {
					t.Error("hourly should be non-zero")
				}
				if got.MonthlyCost == 0 {
					t.Error("monthly should be non-zero")
				}

				// Verify monthly = hourly * HoursPerMonth relationship holds
				if got.MonthlyCost != got.HourlyCost*handler.HoursPerMonth {
					t.Errorf("monthly (%v) should equal hourly*HoursPerMonth (%v)", got.MonthlyCost, got.HourlyCost*handler.HoursPerMonth)
				}
				if got.Status != model.ResourceEstimateStatusUsageEstimated {
					t.Errorf("status = %q, want %q", got.Status, model.ResourceEstimateStatusUsageEstimated)
				}
			} else {
				if got.HourlyCost != tt.wantHourly {
					t.Errorf("hourly = %v, want %v", got.HourlyCost, tt.wantHourly)
				}
				if got.MonthlyCost != tt.wantMonthly {
					t.Errorf("monthly = %v, want %v", got.MonthlyCost, tt.wantMonthly)
				}
				if got.Status != model.ResourceEstimateStatusUsageUnknown {
					t.Errorf("status = %q, want %q", got.Status, model.ResourceEstimateStatusUsageUnknown)
				}
			}
		})
	}
}

func TestDynamoDBHandler_Contract(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryUsageBased
	handlertest.RunContractSuite(t, resourcespec.MustHandler(DynamoDBSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))), handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
			{
				Name:   "provisioned default",
				Region: "us-east-1",
				Attrs:  map[string]any{},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.ProductFamily != "Provisioned IOPS" {
						tb.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Provisioned IOPS")
					}
				},
			},
		},
		DescribeCases: []handlertest.DescribeCase{
			{
				Name:       "nil attrs",
				Attrs:      nil,
				WantAbsent: []string{"billing_mode", "read_capacity", "write_capacity"},
			},
			{
				Name:       "empty attrs",
				Attrs:      map[string]any{},
				WantAbsent: []string{"billing_mode", "read_capacity", "write_capacity"},
			},
			{
				Name: "pay per request",
				Attrs: map[string]any{
					"billing_mode": "PAY_PER_REQUEST",
				},
				WantKeys:   map[string]string{"billing_mode": "PAY_PER_REQUEST"},
				WantAbsent: []string{"read_capacity", "write_capacity"},
			},
			{
				Name: "provisioned with capacity",
				Attrs: map[string]any{
					"billing_mode":   "PROVISIONED",
					"read_capacity":  float64(10),
					"write_capacity": float64(20),
				},
				WantKeys: map[string]string{
					"billing_mode":   "PROVISIONED",
					"read_capacity":  "10",
					"write_capacity": "20",
				},
			},
		},
	})
}

func TestParseDynamoDBAttrs_ParsesStringNumbers(t *testing.T) {
	t.Parallel()

	got := parseDynamoDBAttrs(map[string]any{
		"billing_mode":   "PROVISIONED",
		"read_capacity":  "10",
		"write_capacity": float64(20),
	})

	if got.BillingMode != "PROVISIONED" || got.ReadCapacity != 10 || got.WriteCapacity != 20 {
		t.Fatalf("parseDynamoDBAttrs() = %+v, want PROVISIONED/10/20", got)
	}
}
