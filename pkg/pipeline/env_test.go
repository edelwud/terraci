package pipeline

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
)

func TestModuleEnvVars(t *testing.T) {
	module := discovery.TestModule("payments", "prod", "eu-west-1", "vpc")

	env := ModuleEnvVars(module)

	expected := map[string]string{
		"TF_MODULE_PATH": "payments/prod/eu-west-1/vpc",
		"TF_SERVICE":     "payments",
		"TF_ENVIRONMENT": "prod",
		"TF_REGION":      "eu-west-1",
		"TF_MODULE":      "vpc",
	}

	for key, want := range expected {
		got, ok := env[key]
		if !ok {
			t.Errorf("missing key %s", key)
			continue
		}
		if got != want {
			t.Errorf("env[%s] = %q, want %q", key, got, want)
		}
	}
}
