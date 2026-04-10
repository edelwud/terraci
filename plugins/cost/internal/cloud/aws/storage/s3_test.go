package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestS3Handler_Category(t *testing.T) {
	t.Parallel()

	h := &S3Handler{}
	handlertest.AssertCategory(t, h, handler.CostCategoryUsageBased)
}

func TestS3Handler_NoLookupCapability(t *testing.T) {
	t.Parallel()

	handlertest.AssertNoLookupCapability(t, &S3Handler{})
}

func TestS3Handler_CalculateUsageCost(t *testing.T) {
	t.Parallel()

	h := &S3Handler{}
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

func TestS3Handler_NoDescribeCapability(t *testing.T) {
	t.Parallel()

	handlertest.AssertNoDescribeCapability(t, &S3Handler{})
}
