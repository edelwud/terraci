package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/contracttest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

func mustNewResolver(t testing.TB, runtime contracttest.StubRuntime) *CostResolver {
	t.Helper()
	resolver, err := NewCostResolver(runtime, runtime)
	if err != nil {
		t.Fatalf("NewCostResolver() error = %v", err)
	}
	return resolver
}

func TestCostResolver_ResolveNoProvider(t *testing.T) {
	t.Parallel()

	testRuntime := contracttest.StubRuntime{}
	contracttest.AssertNoProviderContract(t, testRuntime, resourcedef.ResourceType("unknown_resource"))
	resolver := mustNewResolver(t, testRuntime)
	got := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: resourcedef.ResourceType("unknown_resource"),
		Address:      "unknown_resource.example",
	})

	if got.Status != model.ResourceEstimateStatusUnsupported {
		t.Fatalf("Status = %q, want %q", got.Status, model.ResourceEstimateStatusUnsupported)
	}
	if got.FailureKind != model.FailureKindNoProvider {
		t.Fatalf("FailureKind = %q, want %q", got.FailureKind, model.FailureKindNoProvider)
	}
}

func TestCostResolver_ResolveKnownProviderMissingHandler(t *testing.T) {
	t.Parallel()

	testRuntime := contracttest.StubRuntime{
		ResolveProviderFunc: func(resourceType resourcedef.ResourceType) (string, bool) {
			if resourceType == resourcedef.ResourceType("known_without_handler") {
				return "aws", true
			}
			return "", false
		},
	}
	contracttest.AssertNoDefinitionContract(t, testRuntime, "aws", resourcedef.ResourceType("known_without_handler"))
	resolver := mustNewResolver(t, testRuntime)
	got := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: resourcedef.ResourceType("known_without_handler"),
		Address:      "known_without_handler.example",
	})

	if got.Provider != "aws" {
		t.Fatalf("Provider = %q, want %q", got.Provider, "aws")
	}
	if got.Status != model.ResourceEstimateStatusUnsupported {
		t.Fatalf("Status = %q, want %q", got.Status, model.ResourceEstimateStatusUnsupported)
	}
	if got.FailureKind != model.FailureKindNoDefinition {
		t.Fatalf("FailureKind = %q, want %q", got.FailureKind, model.FailureKindNoDefinition)
	}
}

func TestCostResolver_ResolveFixedAndUsageBased(t *testing.T) {
	t.Parallel()

	definitions := map[resourcedef.ResourceType]contracttest.StubDefinition{
		resourcedef.ResourceType("fixed_resource"): {
			CategoryValue: resourcedef.CostCategoryFixed,
			CalculateFixedFunc: func(_ string, _ map[string]any) (hourly, monthly float64) {
				return 0.5, 10
			},
		},
		resourcedef.ResourceType("usage_resource"): {
			CategoryValue: resourcedef.CostCategoryUsageBased,
			CalculateUsageFunc: func(_ string, _ map[string]any) model.UsageCostEstimate {
				return model.UsageCostEstimate{
					HourlyCost:  0.25,
					MonthlyCost: 5,
					Status:      model.ResourceEstimateStatusUsageEstimated,
				}
			},
		},
		resourcedef.ResourceType("usage_unknown_resource"): {
			CategoryValue: resourcedef.CostCategoryUsageBased,
			CalculateUsageFunc: func(_ string, _ map[string]any) model.UsageCostEstimate {
				return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
			},
		},
	}
	resolver := mustNewResolver(t, contracttest.StubRuntime{
		ResolveProviderFunc: func(resourceType resourcedef.ResourceType) (string, bool) {
			if _, ok := definitions[resourceType]; ok {
				return "aws", true
			}
			return "", false
		},
		ResolveDefinitionFunc: func(_ string, resourceType resourcedef.ResourceType) (def resourcedef.Definition, ok bool) {
			stub, ok := definitions[resourceType]
			if !ok {
				return resourcedef.Definition{}, false
			}
			return stub.Definition(resourceType), true
		},
		SourceNameFunc: func(string) string { return "test-source" },
	})

	fixed := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: resourcedef.ResourceType("fixed_resource"),
		Address:      "fixed_resource.example",
	})
	if fixed.MonthlyCost != 10 {
		t.Fatalf("fixed.MonthlyCost = %.2f, want 10", fixed.MonthlyCost)
	}
	if fixed.PriceSource != "fixed" {
		t.Fatalf("fixed.PriceSource = %q, want %q", fixed.PriceSource, "fixed")
	}

	usage := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: resourcedef.ResourceType("usage_resource"),
		Address:      "usage_resource.example",
	})
	if usage.Status != model.ResourceEstimateStatusUsageEstimated {
		t.Fatalf("usage.Status = %q, want %q", usage.Status, model.ResourceEstimateStatusUsageEstimated)
	}
	if usage.PriceSource != "usage-based" {
		t.Fatalf("usage.PriceSource = %q, want %q", usage.PriceSource, "usage-based")
	}
	if usage.MonthlyCost != 5 {
		t.Fatalf("usage.MonthlyCost = %.2f, want 5", usage.MonthlyCost)
	}

	usageUnknown := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: resourcedef.ResourceType("usage_unknown_resource"),
		Address:      "usage_unknown_resource.example",
	})
	if usageUnknown.Status != model.ResourceEstimateStatusUsageUnknown {
		t.Fatalf("usageUnknown.Status = %q, want %q", usageUnknown.Status, model.ResourceEstimateStatusUsageUnknown)
	}
	if usageUnknown.MonthlyCost != 0 {
		t.Fatalf("usageUnknown.MonthlyCost = %.2f, want 0", usageUnknown.MonthlyCost)
	}
}

func TestCostResolver_ResolveUnknownCategory(t *testing.T) {
	t.Parallel()

	resolver := mustNewResolver(t, contracttest.StubRuntime{
		ResolveProviderFunc: func(resourceType resourcedef.ResourceType) (string, bool) {
			if resourceType == resourcedef.ResourceType("weird_resource") {
				return "aws", true
			}
			return "", false
		},
		ResolveDefinitionFunc: func(_ string, resourceType resourcedef.ResourceType) (resourcedef.Definition, bool) {
			if resourceType == resourcedef.ResourceType("weird_resource") {
				return contracttest.StubDefinition{CategoryValue: resourcedef.CostCategory(99)}.Definition(resourceType), true
			}
			return resourcedef.Definition{}, false
		},
	})

	got := resolver.Resolve(context.Background(), ResolveRequest{
		ResourceType: resourcedef.ResourceType("weird_resource"),
		Address:      "weird_resource.example",
	})
	if got.Status != model.ResourceEstimateStatusFailed {
		t.Fatalf("Status = %q, want %q", got.Status, model.ResourceEstimateStatusFailed)
	}
	if got.FailureKind != model.FailureKindInternal {
		t.Fatalf("FailureKind = %q, want %q", got.FailureKind, model.FailureKindInternal)
	}
	if got.StatusDetail == "" {
		t.Fatal("StatusDetail should explain the unknown category")
	}
}

func TestCostResolver_ResolveBeforeCostUnknownCategory(t *testing.T) {
	t.Parallel()

	resolver := mustNewResolver(t, contracttest.StubRuntime{
		ResolveProviderFunc: func(resourceType resourcedef.ResourceType) (string, bool) {
			if resourceType == resourcedef.ResourceType("weird_resource") {
				return "aws", true
			}
			return "", false
		},
		ResolveDefinitionFunc: func(_ string, resourceType resourcedef.ResourceType) (resourcedef.Definition, bool) {
			if resourceType == resourcedef.ResourceType("weird_resource") {
				return contracttest.StubDefinition{CategoryValue: resourcedef.CostCategory(99)}.Definition(resourceType), true
			}
			return resourcedef.Definition{}, false
		},
	})

	rc := model.ResourceCost{}
	resolver.ResolveBeforeCost(context.Background(), &rc, resourcedef.ResourceType("weird_resource"), nil, "us-east-1")
	if rc.Status != model.ResourceEstimateStatusFailed {
		t.Fatalf("Status = %q, want %q", rc.Status, model.ResourceEstimateStatusFailed)
	}
}

func TestCostResolver_ResolveBeforeCostWithState_ReusesPricingIndex(t *testing.T) {
	t.Parallel()

	serviceID := pricing.ServiceID{Provider: "aws", Name: "AmazonEC2"}
	var getIndexCalls int
	testRuntime := contracttest.StubRuntime{
		ResolveProviderFunc: func(resourceType resourcedef.ResourceType) (string, bool) {
			if resourceType == resourcedef.ResourceType("aws_instance") {
				return "aws", true
			}
			return "", false
		},
		ResolveDefinitionFunc: func(_ string, resourceType resourcedef.ResourceType) (resourcedef.Definition, bool) {
			if resourceType != resourcedef.ResourceType("aws_instance") {
				return resourcedef.Definition{}, false
			}
			return contracttest.StubDefinition{
				CategoryValue: resourcedef.CostCategoryStandard,
				LookupFunc: func(region string, _ map[string]any) (*pricing.PriceLookup, error) {
					return &pricing.PriceLookup{
						ServiceID:     serviceID,
						Region:        region,
						ProductFamily: "Compute Instance",
						Attributes: map[string]string{
							"instanceType": "t3.micro",
						},
					}, nil
				},
				CalculateFunc: func(_ *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
					if attrs["instance_type"] == "before" {
						return 1, 10
					}
					return 2, 20
				},
			}.Definition(resourceType), true
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

	resolver := mustNewResolver(t, testRuntime)
	state := NewResolutionState()
	after := resolver.ResolveWithState(context.Background(), ResolveRequest{
		ResourceType: resourcedef.ResourceType("aws_instance"),
		Address:      "aws_instance.web",
		Region:       "us-east-1",
		Attrs:        map[string]any{"instance_type": "after"},
	}, state)

	resolver.ResolveBeforeCostWithState(
		context.Background(),
		&after,
		resourcedef.ResourceType("aws_instance"),
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
