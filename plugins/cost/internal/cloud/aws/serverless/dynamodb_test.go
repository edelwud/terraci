package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestDynamoDBHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &DynamoDBHandler{}

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

			hourly, monthly := h.CalculateCost(nil, nil, "", tt.attrs)

			if tt.expectNonZero {
				if hourly == 0 {
					t.Error("hourly should be non-zero")
				}
				if monthly == 0 {
					t.Error("monthly should be non-zero")
				}

				// Verify monthly = hourly * HoursPerMonth relationship holds
				if monthly != hourly*handler.HoursPerMonth {
					t.Errorf("monthly (%v) should equal hourly*HoursPerMonth (%v)", monthly, hourly*handler.HoursPerMonth)
				}
			} else {
				if hourly != tt.wantHourly {
					t.Errorf("hourly = %v, want %v", hourly, tt.wantHourly)
				}
				if monthly != tt.wantMonthly {
					t.Errorf("monthly = %v, want %v", monthly, tt.wantMonthly)
				}
			}
		})
	}
}

func TestDynamoDBHandler_Contract(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryStandard
	handlertest.RunContractSuite(t, &DynamoDBHandler{}, handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
			{
				Name:   "pay per request",
				Region: "us-east-1",
				Attrs: map[string]any{
					"billing_mode": "PAY_PER_REQUEST",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.ProductFamily != "Amazon DynamoDB PayPerRequest Throughput" {
						tb.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Amazon DynamoDB PayPerRequest Throughput")
					}
				},
			},
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
