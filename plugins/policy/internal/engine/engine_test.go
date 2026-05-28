package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
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

	plan, err := policyinput.NewPlanDocument([]byte(`{"resource_changes":[{"type":"aws_s3_bucket"}]}`))
	if err != nil {
		t.Fatalf("NewPlanDocument() error = %v", err)
	}
	namespaces := policyengine.NewNamespaces([]string{"terraform"})
	envelope := policyinput.NewEnvelope(policyinput.NewTerraCiContext(
		policyinput.NewModuleContext("platform/prod/app", map[string]string{"environment": "prod"}),
		policyinput.NewPolicyContext(namespaces),
		policyinput.NewPlanContext("platform/prod/app/plan.json"),
	), plan)

	evaluation, err := New([]string{policyDir}).Evaluate(context.Background(), envelope, namespaces)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	denies := evaluation.Denies()
	if len(denies) != 1 || denies[0].Message != "prod bucket denied" {
		t.Fatalf("denies = %#v", denies)
	}
	warns := evaluation.Warns()
	if len(warns) != 1 || warns[0].Message != "module warning" {
		t.Fatalf("warns = %#v", warns)
	}
}

func TestEvaluate_NoPolicyFilesReturnsEmptyEvaluation(t *testing.T) {
	t.Parallel()

	plan, err := policyinput.NewPlanDocument([]byte(`{"resource_changes":[]}`))
	if err != nil {
		t.Fatalf("NewPlanDocument() error = %v", err)
	}
	evaluation, err := New([]string{t.TempDir()}).Evaluate(
		context.Background(),
		policyinput.NewEnvelope(policyinput.NewTerraCiContext(
			policyinput.NewModuleContext("platform/prod/app", nil),
			policyinput.NewPolicyContext(policyengine.NewNamespaces([]string{"terraform"})),
			policyinput.NewPlanContext("platform/prod/app/plan.json"),
		), plan),
		policyengine.NewNamespaces([]string{"terraform"}),
	)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if len(evaluation.Denies()) != 0 || len(evaluation.Warns()) != 0 {
		t.Fatalf("evaluation = %#v, want empty", evaluation)
	}
}

func TestParseExpression_ObjectListAndMetadata(t *testing.T) {
	t.Parallel()

	findings := parseExpression([]any{
		"simple finding",
		map[string]any{
			"message":  "object finding",
			"resource": "aws_s3_bucket.logs",
		},
		map[string]any{
			"resource": "aws_s3_bucket.raw",
		},
	}, policyengine.Namespace("terraform"))

	if len(findings) != 3 {
		t.Fatalf("findings = %#v, want 3", findings)
	}
	if findings[0].Message != "simple finding" || findings[0].Namespace != policyengine.Namespace("terraform") {
		t.Fatalf("string finding = %#v", findings[0])
	}
	if findings[1].Message != "object finding" || findings[1].Metadata.Map()["resource"] != "aws_s3_bucket.logs" {
		t.Fatalf("object finding = %#v metadata=%#v", findings[1], findings[1].Metadata.Map())
	}
	if findings[2].Message == "" || findings[2].Metadata.Map()["resource"] != "aws_s3_bucket.raw" {
		t.Fatalf("fallback finding = %#v metadata=%#v", findings[2], findings[2].Metadata.Map())
	}
}
