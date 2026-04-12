package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestS3Handler_Category(t *testing.T) {
	t.Parallel()

	h := resourcespec.MustHandler(S3Spec())
	handlertest.AssertCategory(t, h, handler.CostCategoryUsageBased)
}

func TestS3Handler_BuildLookupReturnsNil(t *testing.T) {
	t.Parallel()

	lookupBuilder, ok := resourcespec.MustHandler(S3Spec()).(handler.LookupBuilder)
	if !ok {
		t.Fatal("handler should implement LookupBuilder")
	}
	handlertest.AssertNilLookup(t, lookupBuilder, "us-east-1", nil)
}

func TestS3Handler_CalculateUsageCost(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(S3Spec()).(handler.UsageBasedCostHandler)
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

func TestS3Handler_DescribeReturnsNil(t *testing.T) {
	t.Parallel()

	describer, ok := resourcespec.MustHandler(S3Spec()).(handler.Describer)
	if !ok {
		t.Fatal("handler should implement Describer")
	}
	if got := describer.Describe(nil, nil); got != nil {
		t.Fatalf("Describe() = %#v, want nil", got)
	}
}
