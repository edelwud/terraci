package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
)

func TestSecretsManagerHandler_Category(t *testing.T) {
	t.Parallel()

	h := &SecretsManagerHandler{}
	handlertest.AssertCategory(t, h, handler.CostCategoryFixed)
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

	handlertest.AssertNilLookup(t, &SecretsManagerHandler{}, "us-east-1", nil)
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
