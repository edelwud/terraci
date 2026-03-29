package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
)

func TestRoute53Handler_Category(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryFixed
	handlertest.RunContractSuite(t, &Route53Handler{}, handlertest.ContractSuite{
		Category:  &category,
		NilLookup: &handlertest.LookupInput{Region: "us-east-1"},
	})
}

func TestRoute53Handler_Describe(t *testing.T) {
	t.Parallel()

	h := &Route53Handler{}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}
}

func TestRoute53Handler_BuildLookup_ReturnsNil(t *testing.T) {
	t.Parallel()

	handlertest.AssertNilLookup(t, &Route53Handler{}, "us-east-1", nil)
}

func TestRoute53Handler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &Route53Handler{}
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)

	if monthly != Route53HostedZoneCost {
		t.Errorf("monthly = %v, want %v", monthly, Route53HostedZoneCost)
	}

	expectedHourly := Route53HostedZoneCost / handler.HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}
