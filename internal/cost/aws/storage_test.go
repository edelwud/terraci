package aws

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestS3BucketHandler_ServiceCode(t *testing.T) {
	h := &S3BucketHandler{}
	if h.ServiceCode() != pricing.ServiceS3 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceS3)
	}
}

func TestS3BucketHandler_BuildLookup(t *testing.T) {
	h := &S3BucketHandler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Errorf("BuildLookup = %v, want nil", lookup)
	}
}

func TestS3BucketHandler_CalculateCost(t *testing.T) {
	h := &S3BucketHandler{}
	hourly, monthly := h.CalculateCost(nil, nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}

func TestCloudWatchLogGroupHandler_ServiceCode(t *testing.T) {
	h := &CloudWatchLogGroupHandler{}
	if h.ServiceCode() != pricing.ServiceCloudWatch {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceCloudWatch)
	}
}

func TestCloudWatchLogGroupHandler_BuildLookup(t *testing.T) {
	h := &CloudWatchLogGroupHandler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Errorf("BuildLookup = %v, want nil", lookup)
	}
}

func TestCloudWatchLogGroupHandler_CalculateCost(t *testing.T) {
	h := &CloudWatchLogGroupHandler{}
	hourly, monthly := h.CalculateCost(nil, nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}

func TestCloudWatchAlarmHandler_ServiceCode(t *testing.T) {
	h := &CloudWatchAlarmHandler{}
	if h.ServiceCode() != pricing.ServiceCloudWatch {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceCloudWatch)
	}
}

func TestCloudWatchAlarmHandler_BuildLookup(t *testing.T) {
	h := &CloudWatchAlarmHandler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}

	if lookup.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", lookup.Region, "us-east-1")
	}

	if lookup.ProductFamily != "Alarm" {
		t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Alarm")
	}

	if lookup.Attributes["location"] != "US East (N. Virginia)" {
		t.Errorf("location = %q, want %q", lookup.Attributes["location"], "US East (N. Virginia)")
	}
}

func TestCloudWatchAlarmHandler_CalculateCost(t *testing.T) {
	h := &CloudWatchAlarmHandler{}

	tests := []struct {
		name        string
		attrs       map[string]any
		wantMonthly float64
	}{
		{
			name:        "no period attr defaults to standard",
			attrs:       nil,
			wantMonthly: CloudWatchStandardAlarmCost,
		},
		{
			name: "standard resolution period=300",
			attrs: map[string]any{
				"period": float64(300),
			},
			wantMonthly: CloudWatchStandardAlarmCost,
		},
		{
			name: "high resolution period=30",
			attrs: map[string]any{
				"period": float64(30),
			},
			wantMonthly: CloudWatchHighResAlarmCost,
		},
		{
			name: "boundary period=60 is standard",
			attrs: map[string]any{
				"period": float64(60),
			},
			wantMonthly: CloudWatchStandardAlarmCost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hourly, monthly := h.CalculateCost(nil, tt.attrs)

			if monthly != tt.wantMonthly {
				t.Errorf("monthly = %v, want %v", monthly, tt.wantMonthly)
			}

			expectedHourly := tt.wantMonthly / HoursPerMonth
			if hourly != expectedHourly {
				t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
			}
		})
	}
}

func TestSecretsManagerHandler_ServiceCode(t *testing.T) {
	h := &SecretsManagerHandler{}
	if h.ServiceCode() != pricing.ServiceSecretsMan {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceSecretsMan)
	}
}

func TestSecretsManagerHandler_CalculateCost(t *testing.T) {
	h := &SecretsManagerHandler{}
	_, monthly := h.CalculateCost(nil, nil)

	if monthly != SecretsManagerSecretCost {
		t.Errorf("monthly = %v, want %v", monthly, SecretsManagerSecretCost)
	}
}

func TestKMSKeyHandler_ServiceCode(t *testing.T) {
	h := &KMSKeyHandler{}
	if h.ServiceCode() != pricing.ServiceKMS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceKMS)
	}
}

func TestKMSKeyHandler_CalculateCost(t *testing.T) {
	h := &KMSKeyHandler{}
	hourly, monthly := h.CalculateCost(nil, nil)

	if monthly != KMSKeyCost {
		t.Errorf("monthly = %v, want %v", monthly, KMSKeyCost)
	}

	expectedHourly := KMSKeyCost / HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}

func TestRoute53ZoneHandler_ServiceCode(t *testing.T) {
	h := &Route53ZoneHandler{}
	if h.ServiceCode() != pricing.ServiceRoute53 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceRoute53)
	}
}

func TestRoute53ZoneHandler_BuildLookup(t *testing.T) {
	h := &Route53ZoneHandler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}

	if lookup.Region != "global" {
		t.Errorf("Region = %q, want %q", lookup.Region, "global")
	}

	if lookup.ProductFamily != "DNS Zone" {
		t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "DNS Zone")
	}
}

func TestRoute53ZoneHandler_CalculateCost(t *testing.T) {
	h := &Route53ZoneHandler{}
	hourly, monthly := h.CalculateCost(nil, nil)

	if monthly != Route53HostedZoneCost {
		t.Errorf("monthly = %v, want %v", monthly, Route53HostedZoneCost)
	}

	expectedHourly := Route53HostedZoneCost / HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}
