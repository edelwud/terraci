package storage

import (
	"testing"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestKMSHandler_ServiceCode(t *testing.T) {
	h := &KMSHandler{}
	if h.ServiceCode() != pricing.ServiceKMS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceKMS)
	}
}

func TestKMSHandler_CalculateCost(t *testing.T) {
	h := &KMSHandler{}
	hourly, monthly := h.CalculateCost(nil, nil)

	if monthly != KMSKeyCost {
		t.Errorf("monthly = %v, want %v", monthly, KMSKeyCost)
	}

	expectedHourly := KMSKeyCost / aws.HoursPerMonth
	if hourly != expectedHourly {
		t.Errorf("hourly = %v, want %v", hourly, expectedHourly)
	}
}
