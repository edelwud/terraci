package elb

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestALBHandler_Category(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryStandard
	handlertest.RunContractSuite(t, resourcespec.MustHandler(ALBSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))), handlertest.ContractSuite{
		Category: &category,
		LookupCases: []handlertest.LookupCase{
			{
				Name:   "default ALB",
				Region: "us-east-1",
				Attrs:  map[string]any{},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.ProductFamily != "Load Balancer-Application" {
						tb.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Load Balancer-Application")
					}
				},
			},
			{
				Name:   "explicit ALB",
				Region: "us-east-1",
				Attrs: map[string]any{
					"load_balancer_type": "application",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.ProductFamily != "Load Balancer-Application" {
						tb.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Load Balancer-Application")
					}
				},
			},
			{
				Name:   "NLB",
				Region: "us-east-1",
				Attrs: map[string]any{
					"load_balancer_type": "network",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.ProductFamily != "Load Balancer-Network" {
						tb.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Load Balancer-Network")
					}
				},
			},
			{
				Name:   "Gateway LB",
				Region: "us-east-1",
				Attrs: map[string]any{
					"load_balancer_type": "gateway",
				},
				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
					tb.Helper()
					if lookup.ProductFamily != "Load Balancer-Gateway" {
						tb.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Load Balancer-Gateway")
					}
				},
			},
		},
		DescribeCases: []handlertest.DescribeCase{
			{
				Name:     "default application",
				Attrs:    map[string]any{},
				WantKeys: map[string]string{"type": "application"},
			},
			{
				Name: "explicit network",
				Attrs: map[string]any{
					"load_balancer_type": "network",
				},
				WantKeys: map[string]string{"type": "network"},
			},
		},
	})
}

func TestALBHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(ALBSpec(awskit.NewRuntimeDeps(awskit.NewRuntime(awskit.Manifest)))).(handler.StandardCostHandler)
	if !ok {
		t.Fatal("handler should implement StandardCostHandler")
	}

	tests := []struct {
		name           string
		price          *pricing.Price
		attrs          map[string]any
		expectedHourly float64
	}{
		{
			name:           "with price",
			price:          &pricing.Price{OnDemandUSD: 0.025},
			attrs:          map[string]any{},
			expectedHourly: 0.025,
		},
		{
			name:           "fallback ALB",
			price:          &pricing.Price{OnDemandUSD: 0},
			attrs:          map[string]any{},
			expectedHourly: 0.0225,
		},
		{
			name:  "fallback NLB",
			price: &pricing.Price{OnDemandUSD: 0},
			attrs: map[string]any{
				"load_balancer_type": "network",
			},
			expectedHourly: 0.0225,
		},
		{
			name:  "fallback GWLB",
			price: &pricing.Price{OnDemandUSD: 0},
			attrs: map[string]any{
				"load_balancer_type": "gateway",
			},
			expectedHourly: 0.0125,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hourly, _ := h.CalculateCost(tt.price, nil, "", tt.attrs)

			if hourly != tt.expectedHourly {
				t.Errorf("hourly = %v, want %v", hourly, tt.expectedHourly)
			}
		})
	}
}

func TestParseLBAttrs_DefaultsToApplication(t *testing.T) {
	t.Parallel()

	got := parseLBAttrs(map[string]any{})
	if got.LoadBalancerType != typeApplication {
		t.Fatalf("parseLBAttrs(default).LoadBalancerType = %q, want %q", got.LoadBalancerType, typeApplication)
	}

	got = parseLBAttrs(map[string]any{"load_balancer_type": typeNetwork})
	if got.LoadBalancerType != typeNetwork {
		t.Fatalf("parseLBAttrs(network).LoadBalancerType = %q, want %q", got.LoadBalancerType, typeNetwork)
	}
}
