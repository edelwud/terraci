package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	policyconfig "github.com/edelwud/terraci/plugins/policy/internal/config"
	"github.com/edelwud/terraci/plugins/policy/internal/domain"
	"github.com/edelwud/terraci/plugins/policy/internal/source"
)

func TestCheck_CIEnforcementFlow(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writePolicyFixture(t, root)
	writePlanFixture(t, root, "platform/prod/eu-central-1/app")
	writePlanFixture(t, root, "platform/sandbox/eu-central-1/app")
	writePlanFixture(t, root, "legacy/old/eu-central-1/db")

	disabled := false
	cfg := &policyconfig.Config{
		Enabled:       true,
		Sources:       []policyconfig.SourceConfig{{Type: policyconfig.SourceTypePath, Path: "policies"}},
		Namespaces:    []string{"terraform"},
		FailureAction: domain.ActionBlock,
		WarningAction: domain.ActionIgnore,
		Overrides: []policyconfig.Override{
			{Match: "**/sandbox/**", FailureAction: domain.ActionWarn},
			{Match: "legacy/**", Enabled: &disabled},
		},
	}
	materializer, err := source.NewMaterializer(cfg, root, filepath.Join(root, ".terraci"))
	if err != nil {
		t.Fatal(err)
	}

	summary, err := Check(context.Background(), CheckRuntime{
		Config:       cfg,
		Sources:      materializer,
		WorkDir:      root,
		PlanSegments: []string{"service", "environment", "region", "module"},
	}, CheckRequest{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if summary.TotalModules != 3 {
		t.Fatalf("TotalModules = %d, want 3", summary.TotalModules)
	}
	if summary.FailedModules != 1 || summary.WarnedModules != 1 || summary.SkippedModules != 1 {
		t.Fatalf("failed/warned/skipped = %d/%d/%d, want 1/1/1", summary.FailedModules, summary.WarnedModules, summary.SkippedModules)
	}
	if summary.TotalSuppressed != 2 {
		t.Fatalf("TotalSuppressed = %d, want 2 warnings ignored", summary.TotalSuppressed)
	}
}

func TestCheck_ModuleFilter(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writePolicyFixture(t, root)
	writePlanFixture(t, root, "platform/prod/eu-central-1/app")
	writePlanFixture(t, root, "platform/sandbox/eu-central-1/app")

	cfg := &policyconfig.Config{
		Enabled:       true,
		Sources:       []policyconfig.SourceConfig{{Type: policyconfig.SourceTypePath, Path: "policies"}},
		Namespaces:    []string{"terraform"},
		FailureAction: domain.ActionWarn,
		WarningAction: domain.ActionIgnore,
	}
	materializer, err := source.NewMaterializer(cfg, root, filepath.Join(root, ".terraci"))
	if err != nil {
		t.Fatal(err)
	}

	summary, err := Check(context.Background(), CheckRuntime{
		Config:       cfg,
		Sources:      materializer,
		WorkDir:      root,
		PlanSegments: []string{"service", "environment", "region", "module"},
	}, CheckRequest{ModulePath: "sandbox/eu-central-1/app"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if summary.TotalModules != 1 || summary.Results[0].Module != "platform/sandbox/eu-central-1/app" {
		t.Fatalf("summary = %#v", summary)
	}
}

func writePolicyFixture(t *testing.T, root string) {
	t.Helper()

	dir := filepath.Join(root, "policies")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	policy := `package terraform
import rego.v1
deny contains msg if {
	input.plan.resource_changes[_].type == "aws_s3_bucket"
	msg := sprintf("bucket denied in %s", [input.terraci.module.path])
}
warn contains msg if {
	input.plan.resource_changes[_].type == "aws_s3_bucket"
	msg := "bucket warning"
}
`
	if err := os.WriteFile(filepath.Join(dir, "terraform.rego"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePlanFixture(t *testing.T, root, modulePath string) {
	t.Helper()

	dir := filepath.Join(root, filepath.FromSlash(modulePath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	plan := `{
		"format_version": "1.0",
		"resource_changes": [{"type": "aws_s3_bucket", "name": "bucket", "change": {"actions": ["create"]}}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
}
