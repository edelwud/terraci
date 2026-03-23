package storage

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestRoute53Handler_Category(t *testing.T) {
	h := &Route53Handler{}
	if h.Category() != aws.CostCategoryFixed {
		t.Errorf("Category() = %v, want CostCategoryFixed", h.Category())
	}
}

func TestRoute53Handler_Describe(t *testing.T) {
	h := &Route53Handler{}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}
}

func TestRoute53Handler_ServiceCode(t *testing.T) {
	h := &Route53Handler{}
	if h.ServiceCode() != pricing.ServiceRoute53 {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceRoute53)
	}
}

func TestRoute53Handler_BuildLookup_ReturnsNil(t *testing.T) {
	h := &Route53Handler{}

	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Error("expected nil lookup for fixed-cost handler")
	}
}

func TestRoute53Handler_CalculateCost(t *testing.T) {
	h := &Route53Handler{}
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)

	if monthly != Route53HostedZoneCost {
		t.Errorf("monthly = %v, want %v", monthly, Route53HostedZoneCost)
	}

	expectedHourly := Route53HostedZoneCost / aws.HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}
