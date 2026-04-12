package elasticache

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestServerlessHandler_Category(t *testing.T) {
	h := resourcespec.MustHandler(ServerlessSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest))))
	handlertest.AssertCategory(t, h, handler.CostCategoryStandard)
}

func TestServerlessHandler_BuildLookup(t *testing.T) {
	h, ok := resourcespec.MustHandler(ServerlessSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.LookupBuilder)
	if !ok {
		t.Fatal("handler should implement LookupBuilder")
	}

	lookup := handlertest.RequireLookup(t, h, "us-east-1", map[string]any{})

	if lookup.ProductFamily != "ElastiCache Serverless" {
		t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "ElastiCache Serverless")
	}
	if lookup.Attributes["usagetype"] != "USE1-ElastiCache:ServerlessStorage" {
		t.Errorf("usagetype = %q, want %q", lookup.Attributes["usagetype"], "USE1-ElastiCache:ServerlessStorage")
	}
}

func TestServerlessHandler_CalculateCost(t *testing.T) {
	h, ok := resourcespec.MustHandler(ServerlessSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.StandardCostHandler)
	if !ok {
		t.Fatal("handler should implement StandardCostHandler")
	}

	price := &pricing.Price{OnDemandUSD: 0.000171} // per GB-hour

	tests := []struct {
		name            string
		attrs           map[string]any
		expectedMonthly float64
	}{
		{
			name:            "default 1GB",
			attrs:           map[string]any{},
			expectedMonthly: 0.000171 * handler.HoursPerMonth,
		},
		{
			name: "10GB configured",
			attrs: map[string]any{
				"cache_usage_limits": []any{
					map[string]any{
						"data_storage": []any{
							map[string]any{
								"maximum": float64(10),
								"unit":    "GB",
							},
						},
					},
				},
			},
			expectedMonthly: 10 * 0.000171 * handler.HoursPerMonth,
		},
	}

	const epsilon = 0.01
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, monthly := h.CalculateCost(price, nil, "", tt.attrs)
			if diff := monthly - tt.expectedMonthly; diff < -epsilon || diff > epsilon {
				t.Errorf("monthly = %v, want %v", monthly, tt.expectedMonthly)
			}
		})
	}
}

func TestServerlessHandler_CalculateCost_FallbackPrice(t *testing.T) {
	h, ok := resourcespec.MustHandler(ServerlessSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.StandardCostHandler)
	if !ok {
		t.Fatal("handler should implement StandardCostHandler")
	}

	// Price with 0 USD — should fall back to default
	price := &pricing.Price{OnDemandUSD: 0}

	attrs := map[string]any{
		"cache_usage_limits": []any{
			map[string]any{
				"data_storage": []any{
					map[string]any{
						"maximum": float64(5),
						"unit":    "GB",
					},
				},
			},
		},
	}

	_, monthly := h.CalculateCost(price, nil, "", attrs)

	expectedMonthly := 5 * FallbackServerlessStorageCostPerGBHour * handler.HoursPerMonth

	const epsilon = 0.01
	if diff := monthly - expectedMonthly; diff < -epsilon || diff > epsilon {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestServerlessHandler_Describe(t *testing.T) {
	h, ok := resourcespec.MustHandler(ServerlessSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.Describer)
	if !ok {
		t.Fatal("handler should implement Describer")
	}

	attrs := map[string]any{
		"engine": "redis",
		"cache_usage_limits": []any{
			map[string]any{
				"data_storage": []any{
					map[string]any{
						"maximum": float64(50),
						"unit":    "GB",
					},
				},
			},
		},
	}

	desc := h.Describe(nil, attrs)
	if desc["type"] != "serverless" {
		t.Errorf("type = %q, want %q", desc["type"], "serverless")
	}
	if desc["engine"] != "redis" {
		t.Errorf("engine = %q, want %q", desc["engine"], "redis")
	}
	if desc["storage_max_gb"] != "50" {
		t.Errorf("storage_max_gb = %q, want %q", desc["storage_max_gb"], "50")
	}
}
