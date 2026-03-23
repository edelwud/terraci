package storage

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestLogGroupHandler_Category(t *testing.T) {
	t.Parallel()

	h := &LogGroupHandler{}
	if h.Category() != aws.CostCategoryUsageBased {
		t.Errorf("Category() = %v, want CostCategoryUsageBased", h.Category())
	}
}

func TestLogGroupHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &LogGroupHandler{}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}
}

func TestAlarmHandler_Category(t *testing.T) {
	t.Parallel()

	h := &AlarmHandler{}
	if h.Category() != aws.CostCategoryFixed {
		t.Errorf("Category() = %v, want CostCategoryFixed", h.Category())
	}
}

func TestLogGroupHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &LogGroupHandler{}
	if h.ServiceCode() != pricing.ServiceCloudWatch {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceCloudWatch)
	}
}

func TestLogGroupHandler_BuildLookup(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	h := &LogGroupHandler{}
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}

func TestAlarmHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &AlarmHandler{}
	if h.ServiceCode() != pricing.ServiceCloudWatch {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceCloudWatch)
	}
}

func TestAlarmHandler_BuildLookup_ReturnsNil(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
			t.Parallel()

			hourly, monthly := h.CalculateCost(nil, nil, "", tt.attrs)

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

func TestAlarmHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &AlarmHandler{}

	tests := []struct {
		name           string
		attrs          map[string]any
		wantResolution string
	}{
		{
			name:           "nil attrs defaults to standard",
			attrs:          nil,
			wantResolution: "standard",
		},
		{
			name:           "no period attr defaults to standard",
			attrs:          map[string]any{},
			wantResolution: "standard",
		},
		{
			name: "high resolution period=30",
			attrs: map[string]any{
				"period": float64(30),
			},
			wantResolution: "high",
		},
		{
			name: "boundary period=60 is standard",
			attrs: map[string]any{
				"period": float64(60),
			},
			wantResolution: "standard",
		},
		{
			name: "standard period=300",
			attrs: map[string]any{
				"period": float64(300),
			},
			wantResolution: "standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := h.Describe(nil, tt.attrs)

			if result["resolution"] != tt.wantResolution {
				t.Errorf("Describe()[resolution] = %q, want %q", result["resolution"], tt.wantResolution)
			}
		})
	}
}
