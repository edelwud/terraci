package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestSecretsManagerHandler_Category(t *testing.T) {
	t.Parallel()

	h := resourcespec.MustHandler(SecretsManagerSpec())
	handlertest.AssertCategory(t, h, handler.CostCategoryFixed)
}

func TestSecretsManagerHandler_CalculateFixedCost(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(SecretsManagerSpec()).(handler.FixedCostHandler)
	if !ok {
		t.Fatal("handler should implement FixedCostHandler")
	}
	_, monthly := h.CalculateFixedCost("", nil)

	if monthly != SecretsManagerSecretCost {
		t.Errorf("monthly = %v, want %v", monthly, SecretsManagerSecretCost)
	}
}

func TestSecretsManagerHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	lookupBuilder, ok := resourcespec.MustHandler(SecretsManagerSpec()).(handler.LookupBuilder)
	if !ok {
		t.Fatal("handler should implement LookupBuilder")
	}
	handlertest.AssertNilLookup(t, lookupBuilder, "us-east-1", nil)
}

func TestSecretsManagerHandler_Describe(t *testing.T) {
	t.Parallel()

	h, ok := resourcespec.MustHandler(SecretsManagerSpec()).(handler.Describer)
	if !ok {
		t.Fatal("handler should implement Describer")
	}
	result := h.Describe(nil, nil)
	if result != nil {
		t.Errorf("Describe() = %v, want nil", result)
	}

	result = h.Describe(nil, map[string]any{"name": "my-secret"})
	if result != nil {
		t.Errorf("Describe() with attrs = %v, want nil", result)
	}
}
