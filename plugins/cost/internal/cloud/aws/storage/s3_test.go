package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
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

func TestS3Handler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &S3Handler{}
	hourly, monthly := h.CalculateCost(nil, nil, "", nil)
	if hourly != 0 {
		t.Errorf("hourly = %v, want 0", hourly)
	}
	if monthly != 0 {
		t.Errorf("monthly = %v, want 0", monthly)
	}
}

func TestS3Handler_NoDescribeCapability(t *testing.T) {
	t.Parallel()

	handlertest.AssertNoDescribeCapability(t, &S3Handler{})
}
