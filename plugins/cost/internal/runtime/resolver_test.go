package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/runtimetest"
)

func TestCostResolver_ResolveNoProvider(t *testing.T) {
	t.Parallel()

	testRuntime := runtimetest.StubRuntime{}
	runtimetest.AssertNoProviderContract(t, testRuntime, handler.ResourceType("unknown_resource"))
	resolver := NewCostResolver(testRuntime)
	got := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: handler.ResourceType("unknown_resource"),
		Address:      "unknown_resource.example",
	})

	if got.ErrorKind != model.CostErrorNoProvider {
		t.Fatalf("ErrorKind = %q, want %q", got.ErrorKind, model.CostErrorNoProvider)
	}
}

func TestCostResolver_ResolveKnownProviderMissingHandler(t *testing.T) {
	t.Parallel()

	testRuntime := runtimetest.StubRuntime{
		ResolveProviderFunc: func(resourceType handler.ResourceType) (string, bool) {
			if resourceType == handler.ResourceType("known_without_handler") {
				return "aws", true
			}
			return "", false
		},
	}
	runtimetest.AssertNoHandlerContract(t, testRuntime, "aws", handler.ResourceType("known_without_handler"))
	resolver := NewCostResolver(testRuntime)
	got := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: handler.ResourceType("known_without_handler"),
		Address:      "known_without_handler.example",
	})

	if got.Provider != "aws" {
		t.Fatalf("Provider = %q, want %q", got.Provider, "aws")
	}
	if got.ErrorKind != model.CostErrorNoHandler {
		t.Fatalf("ErrorKind = %q, want %q", got.ErrorKind, model.CostErrorNoHandler)
	}
}

func TestCostResolver_ResolveFixedAndUsageBased(t *testing.T) {
	t.Parallel()

	handlers := map[handler.ResourceType]handler.ResourceHandler{
		handler.ResourceType("fixed_resource"): runtimetest.StubHandler{
			CategoryValue: handler.CostCategoryFixed,
			CalculateFunc: func(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
				return 0.5, 10
			},
		},
		handler.ResourceType("usage_resource"): runtimetest.StubHandler{CategoryValue: handler.CostCategoryUsageBased},
	}
	resolver := NewCostResolver(runtimetest.StubRuntime{
		ResolveProviderFunc: func(resourceType handler.ResourceType) (string, bool) {
			if _, ok := handlers[resourceType]; ok {
				return "aws", true
			}
			return "", false
		},
		ResolveHandlerFunc: func(_ string, resourceType handler.ResourceType) (handler.ResourceHandler, bool) {
			h, ok := handlers[resourceType]
			return h, ok
		},
		SourceNameFunc: func(string) string { return "test-source" },
	})

	fixed := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: handler.ResourceType("fixed_resource"),
		Address:      "fixed_resource.example",
	})
	if fixed.MonthlyCost != 10 {
		t.Fatalf("fixed.MonthlyCost = %.2f, want 10", fixed.MonthlyCost)
	}
	if fixed.PriceSource != "fixed" {
		t.Fatalf("fixed.PriceSource = %q, want %q", fixed.PriceSource, "fixed")
	}

	usage := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: handler.ResourceType("usage_resource"),
		Address:      "usage_resource.example",
	})
	if usage.ErrorKind != model.CostErrorUsageBased {
		t.Fatalf("usage.ErrorKind = %q, want %q", usage.ErrorKind, model.CostErrorUsageBased)
	}
	if usage.PriceSource != "usage-based" {
		t.Fatalf("usage.PriceSource = %q, want %q", usage.PriceSource, "usage-based")
	}
}

func TestCostResolver_MiddlewareChain(t *testing.T) {
	t.Parallel()

	resolver := NewCostResolver(runtimetest.StubRuntime{
		ResolveProviderFunc: func(resourceType handler.ResourceType) (string, bool) {
			if resourceType == handler.ResourceType("fixed_resource") {
				return "aws", true
			}
			return "", false
		},
		ResolveHandlerFunc: func(_ string, resourceType handler.ResourceType) (handler.ResourceHandler, bool) {
			if resourceType == handler.ResourceType("fixed_resource") {
				return runtimetest.StubHandler{
					CategoryValue: handler.CostCategoryFixed,
					CalculateFunc: func(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
						return 0.5, 10
					},
				}, true
			}
			return nil, false
		},
	})

	var called bool
	resolver.Use(func(ctx context.Context, next ResolveFunc, req ResolveRequest) model.ResourceCost {
		called = true
		rc := next(ctx, req)
		rc.Name = "wrapped"
		return rc
	})

	got := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: handler.ResourceType("fixed_resource"),
		Address:      "fixed_resource.example",
	})

	if !called {
		t.Fatal("middleware was not called")
	}
	if got.Name != "wrapped" {
		t.Fatalf("Name = %q, want %q", got.Name, "wrapped")
	}
}

func TestCostResolver_ResolveBeforeCostWithState_ReusesPricingIndex(t *testing.T) {
	t.Parallel()

	serviceID := pricing.ServiceID{Provider: "aws", Name: "AmazonEC2"}
	var getIndexCalls int
	testRuntime := runtimetest.StubRuntime{
		ResolveProviderFunc: func(resourceType handler.ResourceType) (string, bool) {
			if resourceType == handler.ResourceType("aws_instance") {
				return "aws", true
			}
			return "", false
		},
		ResolveHandlerFunc: func(_ string, resourceType handler.ResourceType) (handler.ResourceHandler, bool) {
			if resourceType != handler.ResourceType("aws_instance") {
				return nil, false
			}
			return lookupHandler{
				StubHandler: runtimetest.StubHandler{
					CategoryValue: handler.CostCategoryStandard,
					CalculateFunc: func(_ *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
						if attrs["instance_type"] == "before" {
							return 1, 10
						}
						return 2, 20
					},
				},
				serviceID: serviceID,
			}, true
		},
		GetIndexFunc: func(_ context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
			getIndexCalls++
			return &pricing.PriceIndex{
				ServiceID: service,
				Region:    region,
				UpdatedAt: time.Now(),
				Products: map[string]pricing.Price{
					"sku": {
						SKU:           "sku",
						ProductFamily: "Compute Instance",
						Attributes:    map[string]string{"instanceType": "t3.micro"},
						OnDemandUSD:   0.01,
					},
				},
			}, nil
		},
		SourceNameFunc: func(string) string { return "test-source" },
	}

	resolver := NewCostResolver(testRuntime)
	state := NewResolutionState()
	after := resolver.ResolveWithState(context.Background(), ResolveRequest{
		ResourceType: handler.ResourceType("aws_instance"),
		Address:      "aws_instance.web",
		Region:       "us-east-1",
		Attrs:        map[string]any{"instance_type": "after"},
	}, state)

	resolver.ResolveBeforeCostWithState(
		context.Background(),
		&after,
		handler.ResourceType("aws_instance"),
		map[string]any{"instance_type": "before"},
		"us-east-1",
		state,
	)

	if getIndexCalls != 1 {
		t.Fatalf("GetIndex() calls = %d, want 1 with shared resolution state", getIndexCalls)
	}
	if after.MonthlyCost != 20 || after.BeforeMonthlyCost != 10 {
		t.Fatalf("after costs = before %.2f / after %.2f, want 10 / 20", after.BeforeMonthlyCost, after.MonthlyCost)
	}
}

type lookupHandler struct {
	runtimetest.StubHandler
	serviceID pricing.ServiceID
}

func (h lookupHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	return &pricing.PriceLookup{
		ServiceID:     h.serviceID,
		Region:        region,
		ProductFamily: "Compute Instance",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
		},
	}, nil
}
