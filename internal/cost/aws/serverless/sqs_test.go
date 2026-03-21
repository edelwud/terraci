package serverless

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestSQSHandler_ServiceCode(t *testing.T) {
	h := &SQSHandler{}
	if h.ServiceCode() != pricing.ServiceSQS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceSQS)
	}
}

func TestSQSHandler_BuildLookup(t *testing.T) {
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
	h := &SQSHandler{}
	hourly, monthly := h.CalculateCost(nil, nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}
