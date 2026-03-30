package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	_ "github.com/edelwud/terraci/plugins/cost/internal/cloud/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/runtimetest"
)

func TestProviderRuntimeRegistry_ResolveProviderAndHandler(t *testing.T) {
	t.Parallel()

	awsProvider, ok := cloud.Get(awskit.ProviderID)
	if !ok {
		t.Fatal("aws provider not registered")
	}

	registry := handler.NewRegistry()
	registry.Register(awskit.ProviderID, handler.ResourceType("aws_instance"), runtimetest.StubHandler{CategoryValue: handler.CostCategoryStandard})

	runtimeRegistry := NewProviderRuntimeRegistry(
		[]cloud.Provider{awsProvider},
		registry,
		map[string]*ProviderRuntime{
			awskit.ProviderID: {
				Definition: awsProvider.Definition(),
				Cache:      pricing.NewCache(t.TempDir(), time.Hour, nil),
			},
		},
	)

	runtimetest.RunResolverRuntimeSuite(t, runtimeRegistry, runtimetest.RuntimeSuite{
		ProviderCases: []runtimetest.ProviderCase{
			{
				Name:         "aws instance provider",
				ResourceType: handler.ResourceType("aws_instance"),
				WantProvider: awskit.ProviderID,
				WantOK:       true,
			},
		},
		HandlerCases: []runtimetest.HandlerCase{
			{
				Name:         "aws instance handler",
				ProviderID:   awskit.ProviderID,
				ResourceType: handler.ResourceType("aws_instance"),
				WantOK:       true,
			},
		},
	})
}

func TestProviderRuntimeRegistry_DistinguishesNoProviderFromNoHandler(t *testing.T) {
	t.Parallel()

	awsProvider, ok := cloud.Get(awskit.ProviderID)
	if !ok {
		t.Fatal("aws provider not registered")
	}

	router := NewResourceProviderRouter()
	router.Register(awskit.ProviderID, handler.ResourceType("aws_cloudfront_distribution"))

	runtimeRegistry := NewProviderRuntimeRegistry(
		[]cloud.Provider{awsProvider},
		handler.NewRegistry(),
		map[string]*ProviderRuntime{
			awskit.ProviderID: {
				Definition: awsProvider.Definition(),
				Cache:      pricing.NewCache(t.TempDir(), time.Hour, nil),
			},
		},
	)
	runtimeRegistry.SetRouter(router)

	runtimetest.AssertNoHandlerContract(t, runtimeRegistry, awskit.ProviderID, handler.ResourceType("aws_cloudfront_distribution"))
	runtimetest.AssertNoProviderContract(t, runtimeRegistry, handler.ResourceType("custom_unknown_resource"))
}

func TestProviderRuntimeRegistry_GetIndexAndSourceName(t *testing.T) {
	t.Parallel()

	awsProvider, ok := cloud.Get(awskit.ProviderID)
	if !ok {
		t.Fatal("aws provider not registered")
	}

	serviceID := awskit.MustService(awskit.ServiceKeyEC2)
	expected := &pricing.PriceIndex{
		ServiceID: serviceID,
		Region:    "us-east-1",
		Version:   "test-v1",
		UpdatedAt: time.Now(),
		Products: map[string]pricing.Price{
			"sku": {SKU: "sku", OnDemandUSD: 0.01},
		},
	}

	runtimeRegistry := NewProviderRuntimeRegistry(
		[]cloud.Provider{awsProvider},
		handler.NewRegistry(),
		map[string]*ProviderRuntime{
			awskit.ProviderID: {
				Definition: awsProvider.Definition(),
				Cache: pricing.NewCache(t.TempDir(), time.Hour, runtimetest.StubFetcher{
					FetchRegionIndexFunc: func(_ context.Context, _ pricing.ServiceID, _ string) (*pricing.PriceIndex, error) {
						return expected, nil
					},
				}),
			},
		},
	)

	got := runtimetest.AssertPricingSourceContract(t, runtimeRegistry, awskit.ProviderID, serviceID, "us-east-1", awsProvider.Definition().Manifest.PriceSource)
	if got.ServiceID != serviceID {
		t.Fatalf("GetIndex().ServiceID = %q, want %q", got.ServiceID, serviceID)
	}
}
