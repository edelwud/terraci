package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
)

func TestLambdaHandler_Category(t *testing.T) {
	t.Parallel()

	handlertest.AssertUsageBasedCategory(t, &LambdaHandler{})
}

func TestLambdaHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &LambdaHandler{}
	lookup := handlertest.RequireLookup(t, h, "us-east-1", nil)

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
	t.Parallel()

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
			t.Parallel()

			hourly, monthly := h.CalculateCost(nil, nil, "", tt.attrs)

			if tt.wantNonZero {
				if hourly == 0 {
					t.Error("hourly should be non-zero")
				}
				if monthly == 0 {
					t.Error("monthly should be non-zero")
				}
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

func TestLambdaHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &LambdaHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantKeys   map[string]string
		wantAbsent []string
	}{
		{
			name:       "nil attrs",
			attrs:      nil,
			wantAbsent: []string{"memory_mb", "runtime", "provisioned_concurrency"},
		},
		{
			name:       "empty attrs",
			attrs:      map[string]any{},
			wantAbsent: []string{"memory_mb", "runtime", "provisioned_concurrency"},
		},
		{
			name: "all fields present",
			attrs: map[string]any{
				"memory_size":                       float64(512),
				"runtime":                           "python3.12",
				"provisioned_concurrent_executions": float64(5),
			},
			wantKeys: map[string]string{
				"memory_mb":               "512",
				"runtime":                 "python3.12",
				"provisioned_concurrency": "5",
			},
		},
		{
			name: "memory and runtime only",
			attrs: map[string]any{
				"memory_size": float64(256),
				"runtime":     "nodejs20.x",
			},
			wantKeys: map[string]string{
				"memory_mb": "256",
				"runtime":   "nodejs20.x",
			},
			wantAbsent: []string{"provisioned_concurrency"},
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
