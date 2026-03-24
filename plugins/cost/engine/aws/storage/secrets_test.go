package storage

import (
	"testing"

	aws "github.com/edelwud/terraci/plugins/cost/engine/aws"
	"github.com/edelwud/terraci/plugins/cost/engine/pricing"
)

func TestSecretsManagerHandler_Category(t *testing.T) {
	t.Parallel()

	h := &SecretsManagerHandler{}
	if h.Category() != aws.CostCategoryFixed {
		t.Errorf("Category() = %v, want CostCategoryFixed", h.Category())
	}
}

func TestSecretsManagerHandler_ServiceCode(t *testing.T) {
	t.Parallel()

	h := &SecretsManagerHandler{}
	if h.ServiceCode() != pricing.ServiceSecretsMan {
		t.Errorf("ServiceCode() = %q, want %q", h.ServiceCode(), pricing.ServiceSecretsMan)
	}
}

func TestSecretsManagerHandler_CalculateCost(t *testing.T) {
	t.Parallel()

	h := &SecretsManagerHandler{}
	_, monthly := h.CalculateCost(nil, nil, "", nil)

	if monthly != SecretsManagerSecretCost {
		t.Errorf("monthly = %v, want %v", monthly, SecretsManagerSecretCost)
	}
}

func TestSecretsManagerHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	h := &SecretsManagerHandler{}
	lookup, err := h.BuildLookup("us-east-1", nil)
	if err != nil {
		t.Fatalf("BuildLookup returned error: %v", err)
	}
	if lookup != nil {
		t.Error("expected nil lookup for fixed-cost handler")
	}
}

func TestSecretsManagerHandler_Describe(t *testing.T) {
	t.Parallel()

	h := &SecretsManagerHandler{}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}

	result = h.Describe(nil, map[string]any{"name": "my-secret"})
	if result != nil {
		t.Errorf("Describe() with attrs = %v, want nil", result)
	}
}
