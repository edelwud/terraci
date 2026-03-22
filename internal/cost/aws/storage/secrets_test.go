package storage

import (
	"testing"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

func TestSecretsManagerHandler_ServiceCode(t *testing.T) {
	h := &SecretsManagerHandler{}
	if h.ServiceCode() != pricing.ServiceSecretsMan {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceSecretsMan)
	}
}

func TestSecretsManagerHandler_CalculateCost(t *testing.T) {
	h := &SecretsManagerHandler{}
	_, monthly := h.CalculateCost(nil, nil, "", nil)

	if monthly != SecretsManagerSecretCost {
		t.Errorf("monthly = %v, want %v", monthly, SecretsManagerSecretCost)
	}
}
