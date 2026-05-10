package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	policyinput "github.com/edelwud/terraci/plugins/policy/internal/input"
)

func TestEvaluate_UsesEnvelopePlanAndTerraciContext(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	policyDir := filepath.Join(dir, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	policy := `package terraform
import rego.v1
deny contains msg if {
	input.plan.resource_changes[_].type == "aws_s3_bucket"
	input.terraci.module.components.environment == "prod"
	msg := "prod bucket denied"
}
warn contains msg if {
	input.terraci.module.path == "platform/prod/app"
	msg := "module warning"
}
`
	if err := os.WriteFile(filepath.Join(policyDir, "policy.rego"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}

	envelope := &policyinput.Envelope{
		TerraCi: policyinput.TerraCiInput{
			Module: policyinput.ModuleInput{
				Path:       "platform/prod/app",
				Components: map[string]string{"environment": "prod"},
			},
			Policy: policyinput.PolicyInput{Namespaces: []string{"terraform"}},
			Plan:   policyinput.PlanInput{Path: "platform/prod/app/plan.json"},
		},
		Plan: map[string]any{"resource_changes": []any{map[string]any{"type": "aws_s3_bucket"}}},
	}

	evaluation, err := New([]string{policyDir}, []string{"terraform"}).Evaluate(context.Background(), envelope)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if len(evaluation.Denies) != 1 || evaluation.Denies[0].Message != "prod bucket denied" {
		t.Fatalf("denies = %#v", evaluation.Denies)
	}
	if len(evaluation.Warns) != 1 || evaluation.Warns[0].Message != "module warning" {
		t.Fatalf("warns = %#v", evaluation.Warns)
	}
}
