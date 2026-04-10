package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
)

func TestKMSHandler_FixedContract(t *testing.T) {
	t.Parallel()

	handlertest.AssertFixedCategory(t, &KMSHandler{})
}

func TestKMSHandler_CalculateFixedCost(t *testing.T) {
	t.Parallel()

	h := &KMSHandler{}
	hourly, monthly := h.CalculateFixedCost("", nil)

	if monthly != KMSKeyCost {
		t.Errorf("monthly = %v, want %v", monthly, KMSKeyCost)
	}

	expectedHourly := KMSKeyCost / handler.HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}

func TestKMSHandler_NoLookupCapability(t *testing.T) {
	t.Parallel()

	handlertest.AssertNoLookupCapability(t, &KMSHandler{})
}

func TestKMSHandler_NoDescribeCapability(t *testing.T) {
	t.Parallel()

	handlertest.AssertNoDescribeCapability(t, &KMSHandler{})
}
