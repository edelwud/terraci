package handlertest

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

func requireDefinition(tb testing.TB, subject any) resourcedef.Definition {
	tb.Helper()

	switch v := subject.(type) {
	case resourcedef.Definition:
		return v
	default:
		tb.Fatalf("unsupported handler test subject type %T; expected resourcedef.Definition", subject)
		return resourcedef.Definition{}
	}
}

// AssertCategory verifies that a definition exposes the expected cost category.
func AssertCategory(tb testing.TB, subject any, want resourcedef.CostCategory) {
	tb.Helper()
	def := requireDefinition(tb, subject)
	if got := def.Category; got != want {
		tb.Fatalf("Category() = %v, want %v", got, want)
	}
}

// AssertNoLookupCapability verifies that a definition does not expose lookup behavior.
func AssertNoLookupCapability(tb testing.TB, subject any) {
	tb.Helper()
	def := requireDefinition(tb, subject)
	if def.Lookup != nil {
		tb.Fatal("definition should not expose lookup behavior")
	}
}

// AssertNoDescribeCapability verifies that a definition does not expose describe behavior.
func AssertNoDescribeCapability(tb testing.TB, subject any) {
	tb.Helper()
	def := requireDefinition(tb, subject)
	if def.Describe != nil {
		tb.Fatal("definition should not expose describe behavior")
	}
}

// RequireLookup builds a lookup and fails the test if the operation errors or returns nil.
func RequireLookup(tb testing.TB, subject any, region string, attrs map[string]any) *pricing.PriceLookup {
	tb.Helper()
	def := requireDefinition(tb, subject)

	lookup, err := def.BuildLookup(region, attrs)
	if err != nil {
		tb.Fatalf("BuildLookup() error = %v", err)
	}
	if lookup == nil {
		tb.Fatal("BuildLookup() returned nil lookup")
	}

	return lookup
}
