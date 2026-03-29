package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
)

func TestDynamoDBHandler_Category(t *testing.T) {
	t.Parallel()

	h := &DynamoDBHandler{}
	if h.Category() != handler.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestDynamoDBHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &DynamoDBHandler{}
	if h.ServiceCode() != awskit.MustService(awskit.ServiceKeyDynamoDB) {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), awskit.MustService(awskit.ServiceKeyDynamoDB))
	}
}

func TestDynamoDBHandler_BuildLookup(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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

func TestDynamoDBHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &DynamoDBHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantKeys   map[string]string
		wantAbsent []string
	}{
		{
			name:       "nil attrs",
			attrs:      nil,
			wantAbsent: []string{"billing_mode", "read_capacity", "write_capacity"},
		},
		{
			name:       "empty attrs",
			attrs:      map[string]any{},
			wantAbsent: []string{"billing_mode", "read_capacity", "write_capacity"},
		},
		{
			name: "pay per request",
			attrs: map[string]any{
				"billing_mode": "PAY_PER_REQUEST",
			},
			wantKeys:   map[string]string{"billing_mode": "PAY_PER_REQUEST"},
			wantAbsent: []string{"read_capacity", "write_capacity"},
		},
		{
			name: "provisioned with capacity",
			attrs: map[string]any{
				"billing_mode":   "PROVISIONED",
				"read_capacity":  float64(10),
				"write_capacity": float64(20),
			},
			wantKeys: map[string]string{
				"billing_mode":   "PROVISIONED",
				"read_capacity":  "10",
				"write_capacity": "20",
			},
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
