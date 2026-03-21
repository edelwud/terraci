package serverless

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestDynamoDBHandler_ServiceCode(t *testing.T) {
	h := &DynamoDBHandler{}
	if h.ServiceCode() != pricing.ServiceDynamoDB {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceDynamoDB)
	}
}

func TestDynamoDBHandler_BuildLookup(t *testing.T) {
	h := &DynamoDBHandler{}

	tests := []struct {
		name              string
		attrs             map[string]any
		wantProductFamily string
	}{
		{
			name: "pay per request",
			attrs: map[string]any{
				"billing_mode": "PAY_PER_REQUEST",
			},
			wantProductFamily: "Amazon DynamoDB PayPerRequest Throughput",
		},
		{
			name:              "provisioned default",
			attrs:             map[string]any{},
			wantProductFamily: "Provisioned IOPS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup, err := h.BuildLookup("us-east-1", tt.attrs)
			if err != nil {
				t.Fatalf("BuildLookup returned error: %v", err)
			}

			if lookup.ProductFamily != tt.wantProductFamily {
				t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, tt.wantProductFamily)
			}
		})
	}
}

func TestDynamoDBHandler_CalculateCost(t *testing.T) {
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
			hourly, monthly := h.CalculateCost(nil, tt.attrs)

			if tt.expectNonZero {
				if hourly == 0 {
					t.Error("hourly should be non-zero")
				}
				if monthly == 0 {
					t.Error("monthly should be non-zero")
				}

				// Verify monthly = hourly * HoursPerMonth relationship holds
				if monthly != hourly*aws.HoursPerMonth {
					t.Errorf("monthly (%v) should equal hourly*HoursPerMonth (%v)", monthly, hourly*aws.HoursPerMonth)
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
