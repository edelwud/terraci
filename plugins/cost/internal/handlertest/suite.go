package handlertest

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// AssertStandardCategory verifies the standard pricing category contract.
func AssertStandardCategory(tb testing.TB, h handler.ResourceHandler) {
	tb.Helper()
	AssertCategory(tb, h, handler.CostCategoryStandard)
}

// AssertFixedCategory verifies the fixed pricing category contract.
func AssertFixedCategory(tb testing.TB, h handler.ResourceHandler) {
	tb.Helper()
	AssertCategory(tb, h, handler.CostCategoryFixed)
}

// AssertUsageBasedCategory verifies the usage-based pricing category contract.
func AssertUsageBasedCategory(tb testing.TB, h handler.ResourceHandler) {
	tb.Helper()
	AssertCategory(tb, h, handler.CostCategoryUsageBased)
}

// AssertFixedContract verifies the common fixed-cost handler contract:
// fixed category and a successful nil lookup.
//
// Example:
//
//	handlertest.AssertFixedContract(t, &Route53Handler{}, "us-east-1", nil)
func AssertFixedContract(tb testing.TB, h interface {
	handler.ResourceHandler
	handler.LookupBuilder
}, region string, attrs map[string]any) {
	tb.Helper()
	AssertFixedCategory(tb, h)
	AssertNilLookup(tb, h, region, attrs)
}

// AssertUsageBasedContract verifies the common usage-based handler contract:
// usage-based category and absence of lookup/describe capabilities.
//
// Example:
//
//	handlertest.AssertUsageBasedContract(t, &SQSHandler{})
func AssertUsageBasedContract(tb testing.TB, h handler.ResourceHandler) {
	tb.Helper()
	AssertUsageBasedCategory(tb, h)
	AssertNoLookupCapability(tb, h)
	AssertNoDescribeCapability(tb, h)
}

// LookupCase defines one contract test case for a handler.LookupBuilder.
type LookupCase struct {
	Name    string
	Region  string
	Attrs   map[string]any
	WantErr bool
	Assert  func(testing.TB, *pricing.PriceLookup)
}

// DescribeCase defines one contract test case for a handler.Describer.
type DescribeCase struct {
	Name       string
	Attrs      map[string]any
	WantKeys   map[string]string
	WantAbsent []string
	Assert     func(testing.TB, map[string]string)
}

// ContractSuite bundles the common handler contract checks into one declarative runner.
//
// Example:
//
//	category := handler.CostCategoryStandard
//	handlertest.RunContractSuite(t, &ClassicHandler{}, handlertest.ContractSuite{
//		Category: &category,
//		LookupCases: []handlertest.LookupCase{
//			{
//				Name:   "default lookup",
//				Region: "us-east-1",
//				Assert: func(tb testing.TB, lookup *pricing.PriceLookup) {
//					tb.Helper()
//					if lookup.ProductFamily != "Load Balancer" {
//						tb.Errorf("ProductFamily = %q, want %q", lookup.ProductFamily, "Load Balancer")
//					}
//				},
//			},
//		},
//		DescribeCases: []handlertest.DescribeCase{
//			{
//				Name:     "default describe",
//				WantKeys: map[string]string{"type": "classic"},
//			},
//		},
//	})
type ContractSuite struct {
	Category         *handler.CostCategory
	ExpectNoLookup   bool
	ExpectNoDescribe bool
	NilLookup        *LookupInput
	LookupCases      []LookupCase
	DescribeCases    []DescribeCase
}

// LookupInput defines one lookup invocation.
type LookupInput struct {
	Region string
	Attrs  map[string]any
}

// AssertNilLookup verifies that BuildLookup succeeds and returns nil.
func AssertNilLookup(tb testing.TB, h handler.LookupBuilder, region string, attrs map[string]any) {
	tb.Helper()

	lookup, err := h.BuildLookup(region, attrs)
	if err != nil {
		tb.Fatalf("BuildLookup() error = %v", err)
	}
	if lookup != nil {
		tb.Fatalf("BuildLookup() = %v, want nil", lookup)
	}
}

// RunLookupCases runs table-driven lookup contract tests with consistent failure messages.
func RunLookupCases(t *testing.T, h handler.LookupBuilder, cases []LookupCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			if tc.WantErr {
				if _, err := h.BuildLookup(tc.Region, tc.Attrs); err == nil {
					t.Fatal("BuildLookup() should return error")
				}
				return
			}

			lookup := RequireLookup(t, h, tc.Region, tc.Attrs)
			if tc.Assert != nil {
				tc.Assert(t, lookup)
			}
		})
	}
}

// RunDescribeCases runs table-driven describe contract tests with common key/absence assertions.
func RunDescribeCases(t *testing.T, h handler.Describer, cases []DescribeCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			result := h.Describe(nil, tc.Attrs)

			for k, want := range tc.WantKeys {
				if got := result[k]; got != want {
					t.Errorf("Describe()[%q] = %q, want %q", k, got, want)
				}
			}
			for _, k := range tc.WantAbsent {
				if _, ok := result[k]; ok {
					t.Errorf("Describe() should not contain key %q", k)
				}
			}
			if tc.Assert != nil {
				tc.Assert(t, result)
			}
		})
	}
}

// RunContractSuite executes the configured handler contract checks.
// It is the preferred entry point for new provider handler tests when multiple
// category/capability/lookup/describe assertions are needed in one place.
func RunContractSuite(t *testing.T, h handler.ResourceHandler, suite ContractSuite) {
	t.Helper()

	if suite.Category != nil {
		AssertCategory(t, h, *suite.Category)
	}
	if suite.ExpectNoLookup {
		AssertNoLookupCapability(t, h)
	}
	if suite.ExpectNoDescribe {
		AssertNoDescribeCapability(t, h)
	}
	if suite.NilLookup != nil {
		lookupBuilder, ok := h.(handler.LookupBuilder)
		if !ok {
			t.Fatal("handler should implement handler.LookupBuilder")
		}
		AssertNilLookup(t, lookupBuilder, suite.NilLookup.Region, suite.NilLookup.Attrs)
	}
	if len(suite.LookupCases) > 0 {
		lookupBuilder, ok := h.(handler.LookupBuilder)
		if !ok {
			t.Fatal("handler should implement handler.LookupBuilder")
		}
		RunLookupCases(t, lookupBuilder, suite.LookupCases)
	}
	if len(suite.DescribeCases) > 0 {
		describer, ok := h.(handler.Describer)
		if !ok {
			t.Fatal("handler should implement handler.Describer")
		}
		RunDescribeCases(t, describer, suite.DescribeCases)
	}
}
