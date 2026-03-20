package aws

import (
	"testing"

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
			hourly, monthly := h.CalculateCost(nil, tt.attrs)

			if tt.wantNonZero {
				if hourly == 0 {
					t.Error("hourly should be non-zero")
				}
				if monthly == 0 {
					t.Error("monthly should be non-zero")
				}
				if monthly != hourly*HoursPerMonth {
					t.Errorf("monthly (%v) should equal hourly*HoursPerMonth (%v)", monthly, hourly*HoursPerMonth)
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

func TestDynamoDBHandler_ServiceCode(t *testing.T) {
	h := &DynamoDBTableHandler{}
	if h.ServiceCode() != pricing.ServiceDynamoDB {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceDynamoDB)
	}
}

func TestDynamoDBHandler_BuildLookup(t *testing.T) {
	h := &DynamoDBTableHandler{}

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
	h := &DynamoDBTableHandler{}

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
				if monthly != hourly*HoursPerMonth {
					t.Errorf("monthly (%v) should equal hourly*HoursPerMonth (%v)", monthly, hourly*HoursPerMonth)
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

func TestSQSQueueHandler_ServiceCode(t *testing.T) {
	h := &SQSQueueHandler{}
	if h.ServiceCode() != pricing.ServiceSQS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceSQS)
	}
}

func TestSQSQueueHandler_BuildLookup(t *testing.T) {
	h := &SQSQueueHandler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Errorf("BuildLookup = %v, want nil", lookup)
	}
}

func TestSQSQueueHandler_CalculateCost(t *testing.T) {
	h := &SQSQueueHandler{}
	hourly, monthly := h.CalculateCost(nil, nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}

func TestSNSTopicHandler_ServiceCode(t *testing.T) {
	h := &SNSTopicHandler{}
	if h.ServiceCode() != pricing.ServiceSNS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceSNS)
	}
}

func TestSNSTopicHandler_BuildLookup(t *testing.T) {
	h := &SNSTopicHandler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Errorf("BuildLookup = %v, want nil", lookup)
	}
}

func TestSNSTopicHandler_CalculateCost(t *testing.T) {
	h := &SNSTopicHandler{}
	hourly, monthly := h.CalculateCost(nil, nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}
