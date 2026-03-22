package elasticache

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestServerlessHandler_ServiceCode(t *testing.T) {
	h := &ServerlessHandler{}
	if h.ServiceCode() != pricing.ServiceElastiCache {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceElastiCache)
	}
}

func TestServerlessHandler_Category(t *testing.T) {
	h := &ServerlessHandler{}
	if h.Category() != aws.CostCategoryStandard {
		t.Errorf("Category() = %v, want CostCategoryStandard", h.Category())
	}
}

func TestServerlessHandler_BuildLookup(t *testing.T) {
	h := &ServerlessHandler{}

	lookup, err := h.BuildLookup("us-east-1", map[string]any{})
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}

	if lookup.ProductFamily != "ElastiCache Serverless" {
		t.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "ElastiCache Serverless")
	}
	if lookup.Attributes["usagetype"] != "USE1-ElastiCache:ServerlessStorage" {
		t.Errorf("usagetype = %q, want %q", lookup.Attributes["usagetype"], "USE1-ElastiCache:ServerlessStorage")
	}
}

func TestServerlessHandler_CalculateCost(t *testing.T) {
	h := &ServerlessHandler{}

	price := &pricing.Price{OnDemandUSD: 0.000171} // per GB-hour

	tests := []struct {
		name            string
		attrs           map[string]any
		expectedMonthly float64
	}{
		{
			name:            "default 1GB",
			attrs:           map[string]any{},
			expectedMonthly: 0.000171 * aws.HoursPerMonth,
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
			expectedMonthly: 10 * 0.000171 * aws.HoursPerMonth,
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
	h := &ServerlessHandler{}

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

	expectedMonthly := 5 * FallbackServerlessStorageCostPerGBHour * aws.HoursPerMonth

	const epsilon = 0.01
	if diff := monthly - expectedMonthly; diff < -epsilon || diff > epsilon {
		t.Errorf("monthly = %v, want %v", monthly, expectedMonthly)
	}
}

func TestServerlessHandler_Describe(t *testing.T) {
	h := &ServerlessHandler{}

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
