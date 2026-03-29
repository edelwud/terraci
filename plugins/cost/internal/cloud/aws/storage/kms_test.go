package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
)

func TestKMSHandler_Category(t *testing.T) {
	t.Parallel()

	h := &KMSHandler{}
	if h.Category() != handler.CostCategoryFixed {
		t.Errorf("Category() = %v, want CostCategoryFixed", h.Category())
	}
}

func TestKMSHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &KMSHandler{}
	if h.ServiceCode() != awskit.MustService(awskit.ServiceKeyKMS) {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), awskit.MustService(awskit.ServiceKeyKMS))
	}
}

func TestKMSHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &KMSHandler{}
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)

	if monthly != KMSKeyCost {
		t.Errorf("monthly = %v, want %v", monthly, KMSKeyCost)
	}

	expectedHourly := KMSKeyCost / handler.HoursPerMonth
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
