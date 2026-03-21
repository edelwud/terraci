package rds

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestClusterHandler_ServiceCode(t *testing.T) {
	h := &ClusterHandler{}
	if h.ServiceCode() != pricing.ServiceRDS {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceRDS)
	}
}
