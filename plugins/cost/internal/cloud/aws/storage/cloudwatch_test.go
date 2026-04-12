package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestLogGroupHandler_Category(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(LogGroupSpec())
	handlertest.AssertCategory(t, def, handler.CostCategoryUsageBased)
}

func TestAlarmHandler_Category(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(AlarmSpec())
	handlertest.AssertCategory(t, def, handler.CostCategoryFixed)
}

func TestLogGroupHandler_CalculateUsageCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(LogGroupSpec())
	got, ok := def.CalculateUsageCost("", nil)
	if !ok {
		t.Fatal("CalculateUsageCost should be available")
	}
	if got.HourlyCost != 0 {
		t.Errorf("hourly = %v, want 0", got.HourlyCost)
	}
	if got.MonthlyCost != 0 {
		t.Errorf("monthly = %v, want 0", got.MonthlyCost)
	}
	if got.Status != model.ResourceEstimateStatusUsageUnknown {
		t.Errorf("status = %q, want %q", got.Status, model.ResourceEstimateStatusUsageUnknown)
	}
}

func TestAlarmHandler_BuildLookup_ReturnsNil(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(AlarmSpec())
	handlertest.AssertNilLookup(t, def, "us-east-1", nil)
}

func TestAlarmHandler_CalculateFixedCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(AlarmSpec())

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

			hourly, monthly, ok := def.CalculateFixedCost("", tt.attrs)
			if !ok {
				t.Fatal("CalculateFixedCost should be available")
			}

			if monthly != tt.wantMonthly {
				t.Errorf("monthly = %v, want %v", monthly, tt.wantMonthly)
			}

			expectedHourly := tt.wantMonthly / handler.HoursPerMonth
			if hourly != expectedHourly {
				t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
			}
		})
	}
}

func TestAlarmHandler_Describe(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(AlarmSpec())

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

			result := def.DescribeResource(nil, tt.attrs)

			if result["resolution"] != tt.wantResolution {
				t.Errorf("DescribeResource()[resolution] = %q, want %q", result["resolution"], tt.wantResolution)
			}
		})
	}
}

func TestParseAlarmAttrs_ParsesStringPeriod(t *testing.T) {
	t.Parallel()

	got := parseAlarmAttrs(map[string]any{"period": "30"})
	if got.Period != 30 {
		t.Fatalf("parseAlarmAttrs().Period = %d, want 30", got.Period)
	}
}
