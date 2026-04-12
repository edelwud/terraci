package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestRoute53Handler_Category(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryFixed
	handlertest.RunContractSuite(t, resourcespec.MustHandler(Route53Spec()), handlertest.ContractSuite{
		Category:  &category,
		NilLookup: &handlertest.LookupInput{Region: "us-east-1"},
	})
}

func TestRoute53Handler_Describe(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(Route53Spec()).(handler.Describer)
	if !ok {
		t.Fatal("handler should implement Describer")
	}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}
}

func TestRoute53Handler_BuildLookup_ReturnsNil(t *testing.T) {
	t.Parallel()

	lookupBuilder, ok := resourcespec.MustHandler(Route53Spec()).(handler.LookupBuilder)
	if !ok {
		t.Fatal("handler should implement LookupBuilder")
	}
	handlertest.AssertNilLookup(t, lookupBuilder, "us-east-1", nil)
}

func TestRoute53Handler_CalculateFixedCost(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(Route53Spec()).(handler.FixedCostHandler)
	if !ok {
		t.Fatal("handler should implement FixedCostHandler")
	}
	hourly, monthly := h.CalculateFixedCost("", nil)

	if monthly != Route53HostedZoneCost {
		t.Errorf("monthly = %v, want %v", monthly, Route53HostedZoneCost)
	}

	expectedHourly := Route53HostedZoneCost / handler.HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}
