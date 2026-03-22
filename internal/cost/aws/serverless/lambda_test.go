package serverless

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestLambdaHandler_ServiceCode(t *testing.T) {
	h := &LambdaHandler{}
	if h.ServiceCode() != pricing.ServiceLambda {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceLambda)
	}
}

func TestLambdaHandler_BuildLookup(t *testing.T) {
	h := &LambdaHandler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}

	if lookup.ProductFamily != "Serverless" {
		t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Serverless")
	}

	if lookup.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", lookup.Region, "us-east-1")
	}

	if lookup.Attributes["location"] != "US East (N. Virginia)" {
		t.Errorf("location = %q, want %q", lookup.Attributes["location"], "US East (N. Virginia)")
	}

	if lookup.Attributes["group"] != "AWS-Lambda-Duration" {
		t.Errorf("group = %q, want %q", lookup.Attributes["group"], "AWS-Lambda-Duration")
	}
}

func TestLambdaHandler_CalculateCost(t *testing.T) {
	h := &LambdaHandler{}

	tests := []struct {
		name        string
		attrs       map[string]any
		wantHourly  float64
		wantMonthly float64
		wantNonZero bool
	}{
		{
			name:        "no provisioned concurrency nil attrs",
			attrs:       nil,
			wantHourly:  0,
			wantMonthly: 0,
		},
		{
			name: "no provisioned concurrency zero value",
			attrs: map[string]any{
				"provisioned_concurrent_executions": float64(0),
			},
			wantHourly:  0,
			wantMonthly: 0,
		},
		{
			name: "with provisioned concurrency and custom memory",
			attrs: map[string]any{
				"provisioned_concurrent_executions": float64(10),
				"memory_size":                       float64(256),
			},
			wantNonZero: true,
		},
		{
			name: "with provisioned concurrency default memory",
			attrs: map[string]any{
				"provisioned_concurrent_executions": float64(10),
				"memory_size":                       float64(0),
			},
			wantNonZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hourly, monthly := h.CalculateCost(nil, nil, "", tt.attrs)

			if tt.wantNonZero {
				if hourly == 0 {
					t.Error("hourly should be non-zero")
				}
				if monthly == 0 {
					t.Error("monthly should be non-zero")
				}
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
