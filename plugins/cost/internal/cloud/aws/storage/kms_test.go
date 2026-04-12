package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestKMSHandler_FixedContract(t *testing.T) {
	t.Parallel()

	handlertest.AssertFixedCategory(t, resourcespec.MustHandler(KMSSpec()))
}

func TestKMSHandler_CalculateFixedCost(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(KMSSpec()).(handler.FixedCostHandler)
	if !ok {
		t.Fatal("handler should implement FixedCostHandler")
	}
	hourly, monthly := h.CalculateFixedCost("", nil)

	if monthly != KMSKeyCost {
		t.Errorf("monthly = %v, want %v", monthly, KMSKeyCost)
	}

	expectedHourly := KMSKeyCost / handler.HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}

func TestKMSHandler_BuildLookupReturnsNil(t *testing.T) {
	t.Parallel()

	lookupBuilder, ok := resourcespec.MustHandler(KMSSpec()).(handler.LookupBuilder)
	if !ok {
		t.Fatal("handler should implement LookupBuilder")
	}
	handlertest.AssertNilLookup(t, lookupBuilder, "us-east-1", nil)
}

func TestKMSHandler_DescribeReturnsNil(t *testing.T) {
	t.Parallel()

	describer, ok := resourcespec.MustHandler(KMSSpec()).(handler.Describer)
	if !ok {
		t.Fatal("handler should implement Describer")
	}
	if got := describer.Describe(nil, nil); got != nil {
		t.Fatalf("Describe() = %#v, want nil", got)
	}
}
