package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestKMSHandler_FixedContract(t *testing.T) {
	t.Parallel()

	handlertest.AssertFixedCategory(t, resourcespec.MustCompileTyped(KMSSpec()))
}

func TestKMSHandler_CalculateFixedCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(KMSSpec())
	hourly, monthly, ok := def.CalculateFixedCost("", nil)
	if !ok {
		t.Fatal("CalculateFixedCost should be available")
	}

	if monthly != KMSKeyCost {
		t.Errorf("monthly = %v, want %v", monthly, KMSKeyCost)
	}

	expectedHourly := KMSKeyCost / costutil.HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}

func TestKMSHandler_BuildLookupReturnsNil(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(KMSSpec())
	handlertest.AssertNilLookup(t, def, "us-east-1", nil)
}

func TestKMSHandler_DescribeReturnsNil(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(KMSSpec())
	if got := def.DescribeResource(nil, nil); got != nil {
		t.Fatalf("DescribeResource() = %#v, want nil", got)
	}
}
