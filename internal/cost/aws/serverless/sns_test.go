package serverless

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestSNSHandler_Category(t *testing.T) {
	h := &SNSHandler{}
	if h.Category() != aws.CostCategoryUsageBased {
		t.Errorf("Category() = %v, want CostCategoryUsageBased", h.Category())
	}
}

func TestSNSHandler_Describe(t *testing.T) {
	h := &SNSHandler{}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}
}

func TestSNSHandler_ServiceCode(t *testing.T) {
	h := &SNSHandler{}
	if h.ServiceCode() != pricing.ServiceSNS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceSNS)
	}
}

func TestSNSHandler_BuildLookup(t *testing.T) {
	h := &SNSHandler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Errorf("BuildLookup = %v, want nil", lookup)
	}
}

func TestSNSHandler_CalculateCost(t *testing.T) {
	h := &SNSHandler{}
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}
