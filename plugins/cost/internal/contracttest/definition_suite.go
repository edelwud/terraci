package contracttest

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

// AssertStandardCategory verifies the standard pricing category contract.
func AssertStandardCategory(tb testing.TB, def resourcedef.Definition) {
	tb.Helper()
	AssertCategory(tb, def, resourcedef.CostCategoryStandard)
}

// AssertFixedCategory verifies the fixed pricing category contract.
func AssertFixedCategory(tb testing.TB, def resourcedef.Definition) {
	tb.Helper()
	AssertCategory(tb, def, resourcedef.CostCategoryFixed)
}

// AssertUsageBasedCategory verifies the usage-based pricing category contract.
func AssertUsageBasedCategory(tb testing.TB, def resourcedef.Definition) {
	tb.Helper()
	AssertCategory(tb, def, resourcedef.CostCategoryUsageBased)
}

// AssertFixedContract verifies the common fixed-cost definition contract:
// fixed category and a successful nil lookup.
//
// Example:
//
//	contracttest.AssertFixedContract(t, def, "us-east-1", nil)
func AssertFixedContract(tb testing.TB, def resourcedef.Definition, region string, attrs map[string]any) {
	tb.Helper()
	AssertFixedCategory(tb, def)
	AssertNilLookup(tb, def, region, attrs)
}

// AssertUsageBasedContract verifies the common usage-based definition contract:
// usage-based category and absence of lookup/describe capabilities.
//
// Example:
//
//	contracttest.AssertUsageBasedContract(t, def)
func AssertUsageBasedContract(tb testing.TB, def resourcedef.Definition) {
	tb.Helper()
	AssertUsageBasedCategory(tb, def)
	AssertNoLookupCapability(tb, def)
	AssertNoDescribeCapability(tb, def)
}

// LookupCase defines one contract test case for a definition lookup function.
type LookupCase struct {
	Name    string
	Region  string
	Attrs   map[string]any
	WantErr bool
	Assert  func(testing.TB, *pricing.PriceLookup)
}

// DescribeCase defines one contract test case for a definition describe function.
type DescribeCase struct {
	Name       string
	Attrs      map[string]any
	WantKeys   map[string]string
	WantAbsent []string
	Assert     func(testing.TB, map[string]string)
}

// ContractSuite bundles common definition contract checks into one declarative runner.
//
// Example:
//
//	category := resourcedef.CostCategoryStandard
//	contracttest.RunContractSuite(t, def, contracttest.ContractSuite{
//		Category: &category,
//		LookupCases: []contracttest.LookupCase{
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
//		DescribeCases: []contracttest.DescribeCase{
//			{
//				Name:     "default describe",
//				WantKeys: map[string]string{"type": "classic"},
//			},
//		},
//	})
type ContractSuite struct {
	Category         *resourcedef.CostCategory
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
func AssertNilLookup(tb testing.TB, def resourcedef.Definition, region string, attrs map[string]any) {
	tb.Helper()

	lookup, err := def.BuildLookup(region, attrs)
	if err != nil {
		tb.Fatalf("BuildLookup() error = %v", err)
	}
	if lookup != nil {
		tb.Fatalf("BuildLookup() = %v, want nil", lookup)
	}
}

// RunLookupCases runs table-driven lookup contract tests with consistent failure messages.
func RunLookupCases(t *testing.T, def resourcedef.Definition, cases []LookupCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			if tc.WantErr {
				if _, err := def.BuildLookup(tc.Region, tc.Attrs); err == nil {
					t.Fatal("BuildLookup() should return error")
				}
				return
			}

			lookup := RequireLookup(t, def, tc.Region, tc.Attrs)
			if tc.Assert != nil {
				tc.Assert(t, lookup)
			}
		})
	}
}

// RunDescribeCases runs table-driven describe contract tests with common key/absence assertions.
func RunDescribeCases(t *testing.T, def resourcedef.Definition, cases []DescribeCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			result := def.DescribeResource(nil, tc.Attrs)

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

// RunContractSuite executes the configured definition contract checks.
// It is the preferred entry point for new provider resource-definition tests when multiple
// category/capability/lookup/describe assertions are needed in one place.
func RunContractSuite(t *testing.T, def resourcedef.Definition, suite ContractSuite) {
	t.Helper()

	if suite.Category != nil {
		AssertCategory(t, def, *suite.Category)
	}
	if suite.ExpectNoLookup {
		AssertNoLookupCapability(t, def)
	}
	if suite.ExpectNoDescribe {
		AssertNoDescribeCapability(t, def)
	}
	if suite.NilLookup != nil {
		AssertNilLookup(t, def, suite.NilLookup.Region, suite.NilLookup.Attrs)
	}
	if len(suite.LookupCases) > 0 {
		if def.Lookup == nil {
			t.Fatal("definition should expose lookup behavior")
		}
		RunLookupCases(t, def, suite.LookupCases)
	}
	if len(suite.DescribeCases) > 0 {
		if def.Describe == nil {
			t.Fatal("definition should expose describe behavior")
		}
		RunDescribeCases(t, def, suite.DescribeCases)
	}
}
