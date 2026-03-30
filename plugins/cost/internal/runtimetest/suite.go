package runtimetest

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// StubRuntime is a compact provider-aware runtime stub for runtime and resolver tests.
type StubRuntime struct {
	ResolveProviderFunc func(resourceType handler.ResourceType) (string, bool)
	ResolveHandlerFunc  func(providerID string, resourceType handler.ResourceType) (handler.ResourceHandler, bool)
	GetIndexFunc        func(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error)
	SourceNameFunc      func(providerID string) string
}

func (r StubRuntime) ResolveProvider(resourceType handler.ResourceType) (string, bool) {
	if r.ResolveProviderFunc == nil {
		return "", false
	}
	return r.ResolveProviderFunc(resourceType)
}

func (r StubRuntime) ResolveHandler(providerID string, resourceType handler.ResourceType) (handler.ResourceHandler, bool) {
	if r.ResolveHandlerFunc == nil {
		return nil, false
	}
	return r.ResolveHandlerFunc(providerID, resourceType)
}

func (r StubRuntime) GetIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
	if r.GetIndexFunc == nil {
		return nil, nil
	}
	return r.GetIndexFunc(ctx, service, region)
}

func (r StubRuntime) SourceName(providerID string) string {
	if r.SourceNameFunc == nil {
		return ""
	}
	return r.SourceNameFunc(providerID)
}

// StubFetcher is a minimal pricing fetcher for cache-backed runtime tests.
type StubFetcher struct {
	FetchRegionIndexFunc func(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error)
}

func (f StubFetcher) FetchRegionIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
	if f.FetchRegionIndexFunc == nil {
		return nil, nil
	}
	return f.FetchRegionIndexFunc(ctx, service, region)
}

// StubHandler is a minimal resource handler for runtime-focused tests.
type StubHandler struct {
	CategoryValue handler.CostCategory
	CalculateFunc func(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64)
}

func (h StubHandler) Category() handler.CostCategory { return h.CategoryValue }

func (h StubHandler) CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
	if h.CalculateFunc != nil {
		return h.CalculateFunc(price, index, region, attrs)
	}
	return 0, 0
}

// ProviderCase defines one contract test case for provider resolution.
type ProviderCase struct {
	Name         string
	ResourceType handler.ResourceType
	WantProvider string
	WantOK       bool
}

// HandlerCase defines one contract test case for handler resolution.
type HandlerCase struct {
	Name         string
	ProviderID   string
	ResourceType handler.ResourceType
	WantOK       bool
	Assert       func(testing.TB, handler.ResourceHandler)
}

// PricingCase defines one contract test case for pricing/source behavior.
type PricingCase struct {
	Name           string
	ServiceID      pricing.ServiceID
	Region         string
	WantErr        bool
	WantNil        bool
	AssertIndex    func(testing.TB, *pricing.PriceIndex)
	SourceProvider string
	WantSource     string
}

// RuntimeSuite groups provider, handler, and pricing contract cases.
type RuntimeSuite struct {
	ProviderCases []ProviderCase
	HandlerCases  []HandlerCase
	PricingCases  []PricingCase
}

type ProviderResolver interface {
	ResolveProvider(resourceType handler.ResourceType) (string, bool)
}

type HandlerResolver interface {
	ResolveHandler(providerID string, resourceType handler.ResourceType) (handler.ResourceHandler, bool)
}

type PricingResolver interface {
	GetIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error)
	SourceName(providerID string) string
}

type ResolverRuntime interface {
	ProviderResolver
	HandlerResolver
	PricingResolver
}

// RunResolverRuntimeSuite executes the configured runtime contract checks.
func RunResolverRuntimeSuite(t *testing.T, runtime ResolverRuntime, suite RuntimeSuite) {
	t.Helper()
	runProviderCases(t, runtime, suite.ProviderCases)
	runHandlerCases(t, runtime, suite.HandlerCases)
	runPricingCases(t, runtime, suite.PricingCases)
}

func runProviderCases(t *testing.T, runtime ResolverRuntime, cases []ProviderCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			gotProvider, gotOK := runtime.ResolveProvider(tc.ResourceType)
			if gotOK != tc.WantOK {
				t.Fatalf("ResolveProvider() ok = %v, want %v", gotOK, tc.WantOK)
			}
			if gotProvider != tc.WantProvider {
				t.Fatalf("ResolveProvider() provider = %q, want %q", gotProvider, tc.WantProvider)
			}
		})
	}
}

func runHandlerCases(t *testing.T, runtime ResolverRuntime, cases []HandlerCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			gotHandler, gotOK := runtime.ResolveHandler(tc.ProviderID, tc.ResourceType)
			if gotOK != tc.WantOK {
				t.Fatalf("ResolveHandler() ok = %v, want %v", gotOK, tc.WantOK)
			}
			if !gotOK {
				return
			}
			if gotHandler == nil {
				t.Fatal("ResolveHandler() returned nil handler")
			}
			if tc.Assert != nil {
				tc.Assert(t, gotHandler)
			}
		})
	}
}

func runPricingCases(t *testing.T, runtime ResolverRuntime, cases []PricingCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			gotIndex, err := runtime.GetIndex(context.Background(), tc.ServiceID, tc.Region)
			if tc.WantErr {
				if err == nil {
					t.Fatal("GetIndex() should return error")
				}
			} else if err != nil {
				t.Fatalf("GetIndex() error = %v", err)
			}

			if tc.WantNil {
				if gotIndex != nil {
					t.Fatalf("GetIndex() = %v, want nil", gotIndex)
				}
			} else if err == nil && gotIndex == nil {
				t.Fatal("GetIndex() returned nil index")
			}

			if tc.AssertIndex != nil && gotIndex != nil {
				tc.AssertIndex(t, gotIndex)
			}

			if tc.SourceProvider != "" || tc.WantSource != "" {
				if gotSource := runtime.SourceName(tc.SourceProvider); gotSource != tc.WantSource {
					t.Fatalf("SourceName() = %q, want %q", gotSource, tc.WantSource)
				}
			}
		})
	}
}

// AssertNoProviderContract verifies that a resource type is not owned by any provider.
func AssertNoProviderContract(tb testing.TB, runtime ProviderResolver, resourceType handler.ResourceType) {
	tb.Helper()

	providerID, ok := runtime.ResolveProvider(resourceType)
	if ok {
		tb.Fatalf("ResolveProvider() = (%q, %v), want no provider", providerID, ok)
	}
}

// AssertNoHandlerContract verifies that a provider owns a type but has no registered handler for it.
func AssertNoHandlerContract(tb testing.TB, runtime interface {
	ProviderResolver
	HandlerResolver
}, providerID string, resourceType handler.ResourceType) {
	tb.Helper()

	gotProvider, ok := runtime.ResolveProvider(resourceType)
	if !ok {
		tb.Fatal("ResolveProvider() should resolve provider ownership")
	}
	if gotProvider != providerID {
		tb.Fatalf("ResolveProvider() = %q, want %q", gotProvider, providerID)
	}
	if h, ok := runtime.ResolveHandler(providerID, resourceType); ok || h != nil {
		tb.Fatalf("ResolveHandler() = (%v, %v), want no handler", h, ok)
	}
}

// AssertPricingSourceContract verifies pricing index lookup and source-name behavior together.
func AssertPricingSourceContract(tb testing.TB, runtime PricingResolver, providerID string, serviceID pricing.ServiceID, region, wantSource string) *pricing.PriceIndex {
	tb.Helper()

	idx, err := runtime.GetIndex(context.Background(), serviceID, region)
	if err != nil {
		tb.Fatalf("GetIndex() error = %v", err)
	}
	if idx == nil {
		tb.Fatal("GetIndex() returned nil index")
	}
	if gotSource := runtime.SourceName(providerID); gotSource != wantSource {
		tb.Fatalf("SourceName() = %q, want %q", gotSource, wantSource)
	}
	return idx
}
