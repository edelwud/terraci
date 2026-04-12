package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/handlertest"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

func TestSecretsManagerHandler_Category(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(SecretsManagerSpec())
	handlertest.AssertCategory(t, def, resourcedef.CostCategoryFixed)
}

func TestSecretsManagerHandler_CalculateFixedCost(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(SecretsManagerSpec())
	_, monthly, ok := def.CalculateFixedCost("", nil)
	if !ok {
		t.Fatal("CalculateFixedCost should be available")
	}

	if monthly != SecretsManagerSecretCost {
		t.Errorf("monthly = %v, want %v", monthly, SecretsManagerSecretCost)
	}
}

func TestSecretsManagerHandler_BuildLookup(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(SecretsManagerSpec())
	handlertest.AssertNilLookup(t, def, "us-east-1", nil)
}

func TestSecretsManagerHandler_Describe(t *testing.T) {
	t.Parallel()

	def := resourcespec.MustCompile(SecretsManagerSpec())
	result := def.DescribeResource(nil, nil)
	if result != nil {
		t.Errorf("DescribeResource() = %v, want nil", result)
	}

	result = def.DescribeResource(nil, map[string]any{"name": "my-secret"})
	if result != nil {
		t.Errorf("DescribeResource() with attrs = %v, want nil", result)
	}
}
