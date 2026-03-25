package storage

import (
	"testing"

	aws "github.com/edelwud/terraci/plugins/cost/internal/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

func TestKMSHandler_Category(t *testing.T) {
	t.Parallel()

	h := &KMSHandler{}
	if h.Category() != aws.CostCategoryFixed {
		t.Errorf("Category() = %v, want CostCategoryFixed", h.Category())
	}
}

func TestKMSHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &KMSHandler{}
	if h.ServiceCode() != pricing.ServiceKMS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceKMS)
	}
}

func TestKMSHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &KMSHandler{}
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)

	if monthly != KMSKeyCost {
		t.Errorf("monthly = %v, want %v", monthly, KMSKeyCost)
	}

	expectedHourly := KMSKeyCost / aws.HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}

func TestKMSHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &KMSHandler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Error("expected nil lookup for fixed-cost handler")
	}
}

func TestKMSHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &KMSHandler{}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}

	result = h.Describe(nil, map[string]any{"key_id": "abc"})
	if result != nil {
		t.Errorf("Describe() with attrs = %v, want nil", result)
	}
}
