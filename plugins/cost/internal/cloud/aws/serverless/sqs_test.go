package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestSQSHandler_Contract(t *testing.T) {
	t.Parallel()

	category := handler.CostCategoryUsageBased
	handlertest.RunContractSuite(t, &SQSHandler{}, handlertest.ContractSuite{
		Category:         &category,
		ExpectNoLookup:   true,
		ExpectNoDescribe: true,
	})
}

func TestSQSHandler_CalculateUsageCost(t *testing.T) {
	t.Parallel()

	h := &SQSHandler{}
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
