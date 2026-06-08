package projectflow

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/pkg/workflow"
)

type testChangeDetector struct {
	changed []*discovery.Module
	baseRef string
}

func (d *testChangeDetector) Name() string        { return "projectflow-test" }
func (d *testChangeDetector) Description() string { return "projectflow test change detector" }

func (d *testChangeDetector) DetectChanges(_ context.Context, req workflow.ChangeDetectionRequest) (*workflow.ChangeDetectionResult, error) {
	d.baseRef = req.BaseRef
	return &workflow.ChangeDetectionResult{Modules: d.changed}, nil
}

func TestRunAdaptsPreparedStateToWorkflowPlanProject(t *testing.T) {
	workDir := t.TempDir()
	cfg := config.Default()
	if err := cfg.Save(filepath.Join(workDir, ".terraci.yaml")); err != nil {
		t.Fatalf("Save config: %v", err)
	}
	stage := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	prod := discovery.TestModule("platform", "prod", "eu-central-1", "vpc")
	for _, module := range []*discovery.Module{stage, prod} {
		dir := filepath.Join(workDir, module.RelativePath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# test\n"), 0o644); err != nil {
			t.Fatalf("write module: %v", err)
		}
	}

	detector := &testChangeDetector{changed: []*discovery.Module{stage}}
	prepared, err := runflow.New(runflow.Options{
		RegistryFactory: func() *registry.Registry {
			return registry.NewFromFactories(func() plugin.Plugin { return detector })
		},
	}).Prepare(context.Background(), runflow.Request{
		CommandName: "projectflow-test",
		WorkDir:     workDir,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	result, err := Run(context.Background(), NewRuntime(prepared), Request{
		Filters: filter.Flags{
			SegmentArgs: []string{"environment=stage"},
		},
		SelectTargets: true,
		ChangedOnly:   true,
		BaseRef:       "origin/main",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if detector.baseRef != "origin/main" {
		t.Fatalf("base ref = %q, want origin/main", detector.baseRef)
	}
	if got := moduleIDs(result.Targets); !reflect.DeepEqual(got, []string{stage.ID()}) {
		t.Fatalf("target ids = %v, want [%s]", got, stage.ID())
	}
}

func moduleIDs(modules []*discovery.Module) []string {
	ids := make([]string, len(modules))
	for i := range modules {
		ids[i] = modules[i].ID()
	}
	return ids
}
