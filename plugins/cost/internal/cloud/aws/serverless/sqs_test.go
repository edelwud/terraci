package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/contracttest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestSQSHandler_Contract(t *testing.T) {
	t.Parallel()

	category := resourcedef.CostCategoryUsageBased
	contracttest.RunContractSuite(t, resourcespec.MustCompileTyped(SQSSpec()), contracttest.ContractSuite{
		Category: &category,
	})
}

func TestSQSHandler_CalculateUsageCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompileTyped(SQSSpec())
	got, ok := def.CalculateUsageCost("", nil)
	if !ok {
		t.Fatal("CalculateUsageCost should be available")
	}
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
