package serverless

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
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
