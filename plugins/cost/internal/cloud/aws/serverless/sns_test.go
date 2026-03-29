package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
)

func TestSNSHandler_UsageBasedContract(t *testing.T) {
	t.Parallel()

	handlertest.AssertUsageBasedContract(t, &SNSHandler{})
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
