package storage

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestLogGroupHandler_ServiceCode(t *testing.T) {
	h := &LogGroupHandler{}
	if h.ServiceCode() != pricing.ServiceCloudWatch {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceCloudWatch)
	}
}

func TestLogGroupHandler_BuildLookup(t *testing.T) {
	h := &LogGroupHandler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Errorf("BuildLookup = %v, want nil", lookup)
	}
}

func TestLogGroupHandler_CalculateCost(t *testing.T) {
	h := &LogGroupHandler{}
	hourly, monthly := h.CalculateCost(nil, nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}

func TestAlarmHandler_ServiceCode(t *testing.T) {
	h := &AlarmHandler{}
	if h.ServiceCode() != pricing.ServiceCloudWatch {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceCloudWatch)
	}
}

func TestAlarmHandler_BuildLookup_ReturnsNil(t *testing.T) {
	h := &AlarmHandler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Error("expected nil lookup for fixed-cost handler")
	}
}

func TestAlarmHandler_CalculateCost(t *testing.T) {
	h := &AlarmHandler{}

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

			expectedHourly := tt.wantMonthly / aws.HoursPerMonth
			if hourly != expectedHourly {
				t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
			}
		})
	}
}
