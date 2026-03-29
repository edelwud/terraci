package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
)

func TestSNSHandler_Category(t *testing.T) {
	t.Parallel()

	h := &SNSHandler{}
	if h.Category() != handler.CostCategoryUsageBased {
		t.Errorf("Category() = %v, want CostCategoryUsageBased", h.Category())
	}
}

func TestSNSHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &SNSHandler{}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}
}

func TestSNSHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &SNSHandler{}
	if h.ServiceCode() != awskit.MustService(awskit.ServiceKeySNS) {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), awskit.MustService(awskit.ServiceKeySNS))
	}
}

func TestSNSHandler_BuildLookup(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	h := &SNSHandler{}
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}
