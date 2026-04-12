package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestS3Handler_Category(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(S3Spec())
	handlertest.AssertCategory(t, def, resourcedef.CostCategoryUsageBased)
}

func TestS3Handler_BuildLookupReturnsNil(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(S3Spec())
	handlertest.AssertNilLookup(t, def, "us-east-1", nil)
}

func TestS3Handler_CalculateUsageCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(S3Spec())
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

func TestS3Handler_DescribeReturnsNil(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(S3Spec())
	if got := def.DescribeResource(nil, nil); got != nil {
		t.Fatalf("DescribeResource() = %#v, want nil", got)
	}
}
