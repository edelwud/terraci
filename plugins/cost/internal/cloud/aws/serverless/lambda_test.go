package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/contracttest"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestLambdaHandler_Category(t *testing.T) {
	t.Parallel()

	contracttest.AssertUsageBasedCategory(t, resourcespec.MustCompileTyped(LambdaSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))))
}

func TestLambdaHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(LambdaSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))
	lookup := contracttest.RequireLookup(t, def, "us-east-1", nil)

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

func TestLambdaHandler_CalculateUsageCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(LambdaSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

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

			got, ok := def.CalculateUsageCost("", tt.attrs)
			if !ok {
				t.Fatal("CalculateUsageCost should be available")
			}

			if tt.wantNonZero {
				if got.HourlyCost == 0 {
					t.Error("hourly should be non-zero")
				}
				if got.MonthlyCost == 0 {
					t.Error("monthly should be non-zero")
				}
				if got.MonthlyCost != got.HourlyCost*costutil.HoursPerMonth {
					t.Errorf("monthly (%v) should equal hourly*HoursPerMonth (%v)", got.MonthlyCost, got.HourlyCost*costutil.HoursPerMonth)
				}
				if got.Status != model.ResourceEstimateStatusUsageEstimated {
					t.Errorf("status = %q, want %q", got.Status, model.ResourceEstimateStatusUsageEstimated)
				}
			} else {
				if got.HourlyCost != tt.wantHourly {
					t.Errorf("hourly = %v, want %v", got.HourlyCost, tt.wantHourly)
				}
				if got.MonthlyCost != tt.wantMonthly {
					t.Errorf("monthly = %v, want %v", got.MonthlyCost, tt.wantMonthly)
				}
				if got.Status != model.ResourceEstimateStatusUsageUnknown {
					t.Errorf("status = %q, want %q", got.Status, model.ResourceEstimateStatusUsageUnknown)
				}
			}
		})
	}
}

func TestLambdaHandler_Describe(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(LambdaSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))

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

			result := def.DescribeResource(nil, tt.attrs)

			for k, v := range tt.wantKeys {
				if result[k] != v {
					t.Errorf("DescribeResource()[%q] = %q, want %q", k, result[k], v)
				}
			}
			for _, k := range tt.wantAbsent {
				if _, ok := result[k]; ok {
					t.Errorf("DescribeResource() should not contain key %q", k)
				}
			}
		})
	}
}
