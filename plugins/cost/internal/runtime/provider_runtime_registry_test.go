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
)

type registryTestFetcher struct {
	index *pricing.PriceIndex
}

func (f registryTestFetcher) FetchRegionIndex(_ context.Context, _ pricing.ServiceID, _ string) (*pricing.PriceIndex, error) {
	return f.index, nil
}

type registryTestHandler struct {
	category handler.CostCategory
}

func (h registryTestHandler) Category() handler.CostCategory { return h.category }

func (registryTestHandler) CalculateCost(*pricing.Price, *pricing.PriceIndex, string, map[string]any) (hourly, monthly float64) {
	return 0, 0
}

func TestProviderRuntimeRegistry_ResolveProviderAndHandler(t *testing.T) {
	t.Parallel()

	awsProvider, ok := cloud.Get(awskit.ProviderID)
	if !ok {
		t.Fatal("aws provider not registered")
	}

	registry := handler.NewRegistry()
	registry.Register(awskit.ProviderID, handler.ResourceType("aws_instance"), registryTestHandler{category: handler.CostCategoryStandard})

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

	providerID, ok := runtimeRegistry.ResolveProvider(handler.ResourceType("aws_instance"))
	if !ok {
		t.Fatal("ResolveProvider should resolve aws_instance")
	}
	if providerID != awskit.ProviderID {
		t.Fatalf("ResolveProvider() = %q, want %q", providerID, awskit.ProviderID)
	}

	resolved, ok := runtimeRegistry.ResolveHandler(providerID, handler.ResourceType("aws_instance"))
	if !ok || resolved == nil {
		t.Fatal("ResolveHandler should resolve a registered handler")
	}
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

	providerID, ok := runtimeRegistry.ResolveProvider(handler.ResourceType("aws_cloudfront_distribution"))
	if !ok {
		t.Fatal("ResolveProvider should resolve known provider-owned resource type")
	}
	if providerID != awskit.ProviderID {
		t.Fatalf("ResolveProvider() = %q, want %q", providerID, awskit.ProviderID)
	}

	if _, ok := runtimeRegistry.ResolveHandler(providerID, handler.ResourceType("aws_cloudfront_distribution")); ok {
		t.Fatal("ResolveHandler should fail when provider is known but no handler is registered")
	}

	if _, ok := runtimeRegistry.ResolveProvider(handler.ResourceType("custom_unknown_resource")); ok {
		t.Fatal("ResolveProvider should fail for an unknown resource type")
	}
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
				Cache:      pricing.NewCache(t.TempDir(), time.Hour, registryTestFetcher{index: expected}),
			},
		},
	)

	got, err := runtimeRegistry.GetIndex(context.Background(), serviceID, "us-east-1")
	if err != nil {
		t.Fatalf("GetIndex() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetIndex() returned nil index")
	}
	if got.ServiceID != serviceID {
		t.Fatalf("GetIndex().ServiceID = %q, want %q", got.ServiceID, serviceID)
	}
	if runtimeRegistry.SourceName(awskit.ProviderID) != awsProvider.Definition().Manifest.PriceSource {
		t.Fatalf("SourceName() = %q, want %q", runtimeRegistry.SourceName(awskit.ProviderID), awsProvider.Definition().Manifest.PriceSource)
	}
}
