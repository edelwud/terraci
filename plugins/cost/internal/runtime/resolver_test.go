package runtime

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

type resolverTestRuntime struct {
	providerID string
	handlers   map[handler.ResourceType]handler.ResourceHandler
}

func (r resolverTestRuntime) ResolveProvider(resourceType handler.ResourceType) (string, bool) {
	if r.providerID == "" {
		return "", false
	}
	_, ok := r.handlers[resourceType]
	if ok {
		return r.providerID, true
	}
	if resourceType == handler.ResourceType("known_without_handler") {
		return r.providerID, true
	}
	return "", false
}

func (r resolverTestRuntime) ResolveHandler(_ string, resourceType handler.ResourceType) (handler.ResourceHandler, bool) {
	h, ok := r.handlers[resourceType]
	return h, ok
}

func (resolverTestRuntime) GetIndex(context.Context, pricing.ServiceID, string) (*pricing.PriceIndex, error) {
	return nil, nil
}

func (resolverTestRuntime) SourceName(string) string {
	return "test-source"
}

type fixedResolverHandler struct{}

func (fixedResolverHandler) Category() handler.CostCategory { return handler.CostCategoryFixed }

func (fixedResolverHandler) CalculateCost(*pricing.Price, *pricing.PriceIndex, string, map[string]any) (hourly, monthly float64) {
	return 0.5, 10
}

type usageResolverHandler struct{}

func (usageResolverHandler) Category() handler.CostCategory { return handler.CostCategoryUsageBased }

func (usageResolverHandler) CalculateCost(*pricing.Price, *pricing.PriceIndex, string, map[string]any) (hourly, monthly float64) {
	return 0, 0
}

func TestCostResolver_ResolveNoProvider(t *testing.T) {
	t.Parallel()

	resolver := NewCostResolver(resolverTestRuntime{})
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

	resolver := NewCostResolver(resolverTestRuntime{providerID: "aws"})
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

	resolver := NewCostResolver(resolverTestRuntime{
		providerID: "aws",
		handlers: map[handler.ResourceType]handler.ResourceHandler{
			handler.ResourceType("fixed_resource"): fixedResolverHandler{},
			handler.ResourceType("usage_resource"): usageResolverHandler{},
		},
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

	resolver := NewCostResolver(resolverTestRuntime{
		providerID: "aws",
		handlers: map[handler.ResourceType]handler.ResourceHandler{
			handler.ResourceType("fixed_resource"): fixedResolverHandler{},
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
