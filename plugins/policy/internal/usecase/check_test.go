package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
	policyinput "github.com/edelwud/terraci/plugins/policy/internal/input"
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
	cfg := &policyengine.Config{
		Enabled:    true,
		Sources:    []policyengine.SourceConfig{{Type: policyengine.SourceTypePath, Path: "policies"}},
		Namespaces: []string{"terraform"},
		Decisions:  policyengine.Decisions{Deny: policyengine.ActionBlock, Warn: policyengine.ActionIgnore},
		Overrides: []policyengine.Override{
			{Match: "**/sandbox/**", Decisions: policyengine.Decisions{Deny: policyengine.ActionWarn}},
			{Match: "legacy/**", Enabled: &disabled},
		},
	}
	materializer, err := source.NewMaterializer(cfg, root, filepath.Join(root, ".terraci"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := Check(context.Background(), CheckRuntime{
		Config:       cfg,
		Sources:      materializer,
		WorkDir:      root,
		PlanSegments: []string{"service", "environment", "region", "module"},
	}, policyengine.CheckRequest{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	summary := result.Summary

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

	cfg := &policyengine.Config{
		Enabled:    true,
		Sources:    []policyengine.SourceConfig{{Type: policyengine.SourceTypePath, Path: "policies"}},
		Namespaces: []string{"terraform"},
		Decisions:  policyengine.Decisions{Deny: policyengine.ActionWarn, Warn: policyengine.ActionIgnore},
	}
	materializer, err := source.NewMaterializer(cfg, root, filepath.Join(root, ".terraci"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := Check(context.Background(), CheckRuntime{
		Config:       cfg,
		Sources:      materializer,
		WorkDir:      root,
		PlanSegments: []string{"service", "environment", "region", "module"},
	}, policyengine.CheckRequest{ModulePath: "sandbox/eu-central-1/app"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	summary := result.Summary
	if summary.TotalModules != 1 || summary.Results[0].Module != "platform/sandbox/eu-central-1/app" {
		t.Fatalf("summary = %#v", summary)
	}
}

func TestCheck_UsesInjectedDependenciesOnce(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writePlanFixture(t, root, "platform/prod/eu-central-1/app")

	materializer := &fakeMaterializer{dirs: []string{filepath.Join(root, "policies")}}
	scanner := fakePlanScanner{collection: &ci.PlanResultCollection{Results: []ci.PlanResult{
		{
			ModulePath: "platform/prod/eu-central-1/app",
			Components: map[string]string{
				"environment": "prod",
			},
		},
	}}}
	evaluator := &fakeEvaluator{
		evaluation: policyengine.NewEvaluation(
			[]policyengine.Finding{{Message: "deny"}},
			nil,
		),
	}
	factory := &fakeEvaluatorFactory{evaluator: evaluator}

	cfg := &policyengine.Config{
		Enabled:    true,
		Sources:    []policyengine.SourceConfig{{Type: policyengine.SourceTypePath, Path: "policies"}},
		Namespaces: []string{"terraform"},
		Decisions:  policyengine.Decisions{Deny: policyengine.ActionWarn},
	}

	result, err := Check(context.Background(), CheckRuntime{
		Config:           cfg,
		Sources:          materializer,
		PlanScanner:      scanner,
		EvaluatorFactory: factory,
		WorkDir:          root,
		PlanSegments:     []string{"service", "environment", "region", "module"},
	}, policyengine.CheckRequest{})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	summary := result.Summary
	if materializer.calls != 1 {
		t.Fatalf("materializer calls = %d, want 1", materializer.calls)
	}
	if factory.calls != 1 {
		t.Fatalf("evaluator factory calls = %d, want 1", factory.calls)
	}
	if evaluator.calls != 1 {
		t.Fatalf("evaluator calls = %d, want 1", evaluator.calls)
	}
	if evaluator.namespaces[0] != policyengine.Namespace("terraform") {
		t.Fatalf("namespaces = %v, want [terraform]", evaluator.namespaces)
	}
	if got := evaluator.inputs[0].OPAValue()["terraci"].(map[string]any)["module"].(map[string]any)["path"]; got != "platform/prod/eu-central-1/app" {
		t.Fatalf("evaluator input module path = %v", got)
	}
	if summary.WarnedModules != 1 || summary.FailedModules != 0 {
		t.Fatalf("warned/failed = %d/%d, want 1/0", summary.WarnedModules, summary.FailedModules)
	}
	if result.PlanResults != scanner.collection {
		t.Fatal("PlanResults did not preserve scanner collection")
	}
}

type fakeMaterializer struct {
	dirs  []string
	calls int
}

func (f *fakeMaterializer) Materialize(_ context.Context, _ string) ([]string, error) {
	f.calls++
	return append([]string(nil), f.dirs...), nil
}

func (f *fakeMaterializer) CacheDir(string) string {
	return ""
}

type fakePlanScanner struct {
	collection *ci.PlanResultCollection
}

func (f fakePlanScanner) Scan(string, []string) (*ci.PlanResultCollection, error) {
	return f.collection, nil
}

type fakeEvaluatorFactory struct {
	evaluator *fakeEvaluator
	calls     int
}

func (f *fakeEvaluatorFactory) NewEvaluator([]string) Evaluator {
	f.calls++
	return f.evaluator
}

type fakeEvaluator struct {
	evaluation *policyengine.Evaluation
	inputs     []policyinput.Envelope
	namespaces policyengine.Namespaces
	calls      int
}

func (f *fakeEvaluator) Evaluate(_ context.Context, input policyinput.Envelope, namespaces policyengine.Namespaces) (*policyengine.Evaluation, error) {
	f.calls++
	f.inputs = append(f.inputs, input)
	f.namespaces = namespaces.Clone()
	return f.evaluation, nil
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
