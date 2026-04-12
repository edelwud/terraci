package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestSQSHandler_Contract(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryUsageBased
	handlertest.RunContractSuite(t, resourcespec.MustHandler(SQSSpec()), handlertest.ContractSuite{
		Category: &category,
	})
}

func TestSQSHandler_CalculateUsageCost(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(SQSSpec()).(handler.UsageBasedCostHandler)
	if !ok {
		t.Fatal("handler should implement UsageBasedCostHandler")
	}
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
