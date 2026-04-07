package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	_ "github.com/edelwud/terraci/plugins/cost/internal/cloud/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/runtimetest"
	"github.com/edelwud/terraci/plugins/diskblob"
)

func TestProviderCatalog_ResolveProviderAndHandler(t *testing.T) {
	t.Parallel()

	registry := handler.NewRegistry()
	registry.Register(awskit.ProviderID, handler.ResourceType("aws_instance"), runtimetest.StubHandler{CategoryValue: handler.CostCategoryStandard})

	catalog := NewProviderCatalog(
		funcProviderRouter(func(resourceType handler.ResourceType) (string, bool) {
			if resourceType == handler.ResourceType("aws_instance") {
				return awskit.ProviderID, true
			}
			return "", false
		}),
		registry,
		map[string]model.ProviderMetadata{
			awskit.ProviderID: {DisplayName: "AWS", PriceSource: "aws-bulk-api"},
		},
	)

	if providerID, ok := catalog.ResolveProvider(handler.ResourceType("aws_instance")); !ok || providerID != awskit.ProviderID {
		t.Fatalf("ResolveProvider() = (%q, %v), want (%q, true)", providerID, ok, awskit.ProviderID)
	}
	if h, ok := catalog.ResolveHandler(awskit.ProviderID, handler.ResourceType("aws_instance")); !ok || h == nil {
		t.Fatal("ResolveHandler() should resolve aws_instance")
	}

	if meta := catalog.ProviderMetadata(); meta[awskit.ProviderID].PriceSource != "aws-bulk-api" {
		t.Fatalf("ProviderMetadata().PriceSource = %q, want %q", meta[awskit.ProviderID].PriceSource, "aws-bulk-api")
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
		map[string]*ProviderRuntime{
			awskit.ProviderID: {
				Definition: awsProvider.Definition(),
				Cache: pricing.NewCache(diskblob.NewStore(t.TempDir()), "", time.Hour, runtimetest.StubFetcher{
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

func TestProviderCatalog_DistinguishesNoProviderFromNoHandler(t *testing.T) {
	t.Parallel()

	awsProvider, ok := cloud.Get(awskit.ProviderID)
	if !ok {
		t.Fatal("aws provider not registered")
	}

	router := NewResourceProviderRouter()
	router.Register(awskit.ProviderID, handler.ResourceType("aws_cloudfront_distribution"))

	catalog := NewProviderCatalog(router, handler.NewRegistry(), map[string]model.ProviderMetadata{
		awskit.ProviderID: {
			DisplayName: awsProvider.Definition().Manifest.DisplayName,
			PriceSource: awsProvider.Definition().Manifest.PriceSource,
		},
	})

	runtimetest.AssertNoHandlerContract(t, catalog, awskit.ProviderID, handler.ResourceType("aws_cloudfront_distribution"))
	runtimetest.AssertNoProviderContract(t, catalog, handler.ResourceType("custom_unknown_resource"))
}

func TestProviderRuntimeRegistry_WarmIndexes(t *testing.T) {
	t.Parallel()

	awsProvider, ok := cloud.Get(awskit.ProviderID)
	if !ok {
		t.Fatal("aws provider not registered")
	}

	serviceID := awskit.MustService(awskit.ServiceKeyEC2)
	fetchCount := 0
	runtimeRegistry := NewProviderRuntimeRegistry(
		map[string]*ProviderRuntime{
			awskit.ProviderID: {
				Definition: awsProvider.Definition(),
				Cache: pricing.NewCache(diskblob.NewStore(t.TempDir()), "", time.Hour, runtimetest.StubFetcher{
					FetchRegionIndexFunc: func(_ context.Context, _ pricing.ServiceID, _ string) (*pricing.PriceIndex, error) {
						fetchCount++
						return &pricing.PriceIndex{
							ServiceID: serviceID,
							Region:    "us-east-1",
							Version:   "test-v1",
							UpdatedAt: time.Now(),
							Products: map[string]pricing.Price{
								"sku": {SKU: "sku", OnDemandUSD: 0.01},
							},
						}, nil
					},
				}),
			},
		},
	)

	services := map[pricing.ServiceID][]string{serviceID: {"us-east-1"}}
	if err := runtimeRegistry.WarmIndexes(context.Background(), services); err != nil {
		t.Fatalf("WarmIndexes() error = %v", err)
	}
	if fetchCount != 1 {
		t.Fatalf("fetchCount = %d, want 1", fetchCount)
	}
	// Second warm-up should use the cached index, not fetch again.
	if err := runtimeRegistry.WarmIndexes(context.Background(), services); err != nil {
		t.Fatalf("second WarmIndexes() error = %v", err)
	}
	if fetchCount != 1 {
		t.Fatalf("fetchCount after cached warm = %d, want 1", fetchCount)
	}
}

type funcProviderRouter func(resourceType handler.ResourceType) (string, bool)

func (f funcProviderRouter) ResolveProvider(resourceType handler.ResourceType) (string, bool) {
	return f(resourceType)
}
