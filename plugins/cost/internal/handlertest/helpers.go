package handlertest

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// AssertCategory verifies that a handler exposes the expected cost category.
func AssertCategory(tb testing.TB, h handler.ResourceHandler, want handler.CostCategory) {
	tb.Helper()
	if got := h.Category(); got != want {
		tb.Fatalf("Category() = %v, want %v", got, want)
	}
}

// AssertNoLookupCapability verifies that a handler does not implement LookupBuilder.
func AssertNoLookupCapability(tb testing.TB, h any) {
	tb.Helper()
	if _, ok := h.(handler.LookupBuilder); ok {
		tb.Fatal("handler should not implement handler.LookupBuilder")
	}
}

// AssertNoDescribeCapability verifies that a handler does not implement Describer.
func AssertNoDescribeCapability(tb testing.TB, h any) {
	tb.Helper()
	if _, ok := h.(handler.Describer); ok {
		tb.Fatal("handler should not implement handler.Describer")
	}
}

// RequireLookup builds a lookup and fails the test if the operation errors or returns nil.
func RequireLookup(tb testing.TB, h handler.LookupBuilder, region string, attrs map[string]any) *pricing.PriceLookup {
	tb.Helper()

	lookup, err := h.BuildLookup(region, attrs)
	if err != nil {
		tb.Fatalf("BuildLookup() error = %v", err)
	}
	if lookup == nil {
		tb.Fatal("BuildLookup() returned nil lookup")
	}

	return lookup
}
