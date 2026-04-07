package runtime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
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

	router := NewResourceProviderRouter()
	router.Register(awskit.ProviderID, handler.ResourceType("aws_instance"))

	catalog := NewProviderCatalog(
		router,
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

	cache1 := pricing.NewCacheFromBlobCache(blobcache.New(diskblob.NewStore(t.TempDir()), "", time.Hour))
	cache1.SetFetcher(runtimetest.StubFetcher{
		FetchRegionIndexFunc: func(_ context.Context, _ pricing.ServiceID, _ string) (*pricing.PriceIndex, error) {
			return expected, nil
		},
	})
	runtimeRegistry := NewProviderRuntimeRegistry(
		map[string]*ProviderRuntime{
			awskit.ProviderID: {
				Definition: awsProvider.Definition(),
				Cache:      cache1,
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
	cache2 := pricing.NewCacheFromBlobCache(blobcache.New(diskblob.NewStore(t.TempDir()), "", time.Hour))
	cache2.SetFetcher(runtimetest.StubFetcher{
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
	})
	runtimeRegistry := NewProviderRuntimeRegistry(
		map[string]*ProviderRuntime{
			awskit.ProviderID: {
				Definition: awsProvider.Definition(),
				Cache:      cache2,
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

func TestProviderRuntimeRegistry_SharedFetcherOverrideDoesNotPanicForMultipleProviders(t *testing.T) {
	t.Parallel()

	awsProvider, ok := cloud.Get(awskit.ProviderID)
	if !ok {
		t.Fatal("aws provider not registered")
	}
	providers := []cloud.Provider{
		awsProvider,
		fakeCloudProvider{id: "fake", serviceName: "FakeService"},
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("constructor should not panic for multi-provider setup: %v", recovered)
		}
	}()
	_ = NewProviderRuntimeRegistryFromProviders(
		providers,
		blobcache.New(diskblob.NewStore(t.TempDir()), "", time.Hour),
		nil,
	)
}

func TestProviderRuntimeRegistry_ProviderScopedFetcherOverrides(t *testing.T) {
	t.Parallel()

	providers := []cloud.Provider{
		fakeCloudProvider{id: "one", serviceName: "ServiceOne"},
		fakeCloudProvider{id: "two", serviceName: "ServiceTwo"},
	}
	serviceOne := pricing.ServiceID{Provider: "one", Name: "ServiceOne"}
	serviceTwo := pricing.ServiceID{Provider: "two", Name: "ServiceTwo"}

	runtimeRegistry := NewProviderRuntimeRegistryFromProviders(
		providers,
		blobcache.New(diskblob.NewStore(t.TempDir()), "", time.Hour),
		map[string]pricing.PriceFetcher{
			"one": runtimetest.StubFetcher{
				FetchRegionIndexFunc: func(_ context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
					return testPriceIndex(service, region, "override-one"), nil
				},
			},
			"two": runtimetest.StubFetcher{
				FetchRegionIndexFunc: func(_ context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
					return testPriceIndex(service, region, "override-two"), nil
				},
			},
		},
	)

	gotOne, err := runtimeRegistry.GetIndex(context.Background(), serviceOne, "us-east-1")
	if err != nil {
		t.Fatalf("GetIndex(one) error = %v", err)
	}
	if gotOne.Version != "override-one" {
		t.Fatalf("GetIndex(one).Version = %q, want override-one", gotOne.Version)
	}

	gotTwo, err := runtimeRegistry.GetIndex(context.Background(), serviceTwo, "us-east-1")
	if err != nil {
		t.Fatalf("GetIndex(two) error = %v", err)
	}
	if gotTwo.Version != "override-two" {
		t.Fatalf("GetIndex(two).Version = %q, want override-two", gotTwo.Version)
	}
}

type fakeCloudProvider struct {
	id          string
	serviceName string
}

func (p fakeCloudProvider) Definition() cloud.Definition {
	service := pricing.ServiceID{Provider: p.id, Name: p.serviceName}
	return cloud.Definition{
		ConfigKey: p.id,
		Manifest: pricing.ProviderManifest{
			ID:          p.id,
			DisplayName: p.id,
			PriceSource: "test",
			Services: pricing.ServiceCatalog{
				"default": service,
			},
		},
		FetcherFactory: func() pricing.PriceFetcher {
			return runtimetest.StubFetcher{
				FetchRegionIndexFunc: func(_ context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
					return testPriceIndex(service, region, fmt.Sprintf("default-%s", p.id)), nil
				},
			}
		},
	}
}

func testPriceIndex(service pricing.ServiceID, region, version string) *pricing.PriceIndex {
	return &pricing.PriceIndex{
		ServiceID: service,
		Region:    region,
		Version:   version,
		UpdatedAt: time.Now(),
		Products: map[string]pricing.Price{
			"sku": {SKU: "sku", OnDemandUSD: 0.01},
		},
	}
}
