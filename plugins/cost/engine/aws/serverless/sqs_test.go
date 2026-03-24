package serverless

import (
	"testing"

	aws "github.com/edelwud/terraci/plugins/cost/engine/aws"
	"github.com/edelwud/terraci/plugins/cost/engine/pricing"
)

func TestSQSHandler_Category(t *testing.T) {
	t.Parallel()

	h := &SQSHandler{}
	if h.Category() != aws.CostCategoryUsageBased {
		t.Errorf("Category() = %v, want CostCategoryUsageBased", h.Category())
	}
}

func TestSQSHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &SQSHandler{}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}
}

func TestSQSHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &SQSHandler{}
	if h.ServiceCode() != pricing.ServiceSQS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceSQS)
	}
}

func TestSQSHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &SQSHandler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Errorf("BuildLookup = %v, want nil", lookup)
	}
}

func TestSQSHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &SQSHandler{}
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}
