package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestSNSHandler_UsageBasedContract(t *testing.T) {
	t.Parallel()

	handlertest.AssertUsageBasedContract(t, &SNSHandler{})
}

func TestSNSHandler_CalculateUsageCost(t *testing.T) {
	t.Parallel()

	h := &SNSHandler{}
	got := h.CalculateUsageCost("", nil)
	if got.HourlyCost != 0 {
		t.Errorf("hourly = %v, want 0", got.HourlyCost)
	}
	if got.MonthlyCost != 0 {
		t.Errorf("monthly = %v, want 0", got.MonthlyCost)
	}
	if got.Status != model.ResourceEstimateStatusUsageUnknown {
		t.Errorf("status = %q, want %q", got.Status, model.ResourceEstimateStatusUsageUnknown)
	}
}
