package generateflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/pkg/workflow"
)

type testCIProvider struct{}

func (testCIProvider) Name() string        { return "test-ci" }
func (testCIProvider) Description() string { return "test CI provider" }
func (testCIProvider) DetectEnv() bool     { return false }
func (testCIProvider) ProviderName() string {
	return "test"
}
func (testCIProvider) PipelineID() string { return "" }
func (testCIProvider) CommitSHA() string  { return "" }
func (testCIProvider) PipelineRequirements(*plugin.AppContext) pipeline.BuildRequirements {
	return pipeline.BuildRequirements{}
}
func (testCIProvider) NewGenerator(_ *plugin.AppContext, ir *pipeline.IR) pipeline.Generator {
	return testGenerator{ir: ir}
}

type testGenerator struct {
	ir *pipeline.IR
}

func (g testGenerator) Generate() (pipeline.GeneratedPipeline, error) {
	return testGeneratedPipeline{}, nil
}

func (g testGenerator) DryRun() (*pipeline.DryRunResult, error) {
	return g.ir.DryRun(g.ir.ModuleCount()), nil
}

type testGeneratedPipeline struct{}

func (testGeneratedPipeline) ToYAML() ([]byte, error) {
	return []byte("test: true\n"), nil
}

type noChangeDetector struct{}

func (noChangeDetector) Name() string        { return "no-change" }
func (noChangeDetector) Description() string { return "no change detector" }
func (noChangeDetector) DetectChanges(context.Context, workflow.ChangeDetectionRequest) (*workflow.ChangeDetectionResult, error) {
	return &workflow.ChangeDetectionResult{}, nil
}

func TestRunDryRun(t *testing.T) {
	workDir := testProject(t)
	prepared := prepareGenerate(t, workDir, func() plugin.Plugin { return testCIProvider{} })

	result, err := Run(context.Background(), NewRuntime(prepared), Request{DryRun: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.DryRun == nil {
		t.Fatal("DryRun = nil")
	}
	if result.DryRun.AffectedModules != 1 {
		t.Fatalf("AffectedModules = %d, want 1", result.DryRun.AffectedModules)
	}
	if result.Pipeline != nil {
		t.Fatal("Pipeline should be nil for dry-run")
	}
}

func TestRunGenerate(t *testing.T) {
	workDir := testProject(t)
	prepared := prepareGenerate(t, workDir, func() plugin.Plugin { return testCIProvider{} })

	result, err := Run(context.Background(), NewRuntime(prepared), Request{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Pipeline == nil {
		t.Fatal("Pipeline = nil")
	}
	if result.Skipped {
		t.Fatal("Skipped = true")
	}
}

func TestRunChangedOnlyNoTargetsSkipsBeforeProviderResolution(t *testing.T) {
	workDir := testProject(t)
	prepared := prepareGenerate(t, workDir, func() plugin.Plugin { return noChangeDetector{} })

	result, err := Run(context.Background(), NewRuntime(prepared), Request{ChangedOnly: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !result.Skipped {
		t.Fatal("Skipped = false, want true")
	}
	if result.Pipeline != nil || result.DryRun != nil {
		t.Fatalf("unexpected generation result: pipeline=%v dryRun=%v", result.Pipeline, result.DryRun)
	}
}

func prepareGenerate(tb testing.TB, workDir string, factories ...registry.Factory) *runflow.Prepared {
	tb.Helper()
	prepared, err := runflow.New(runflow.Options{
		RegistryFactory: func() *registry.Registry {
			return registry.NewFromFactories(factories...)
		},
	}).Prepare(context.Background(), runflow.Request{
		CommandName: "generateflow-test",
		WorkDir:     workDir,
	})
	if err != nil {
		tb.Fatalf("Prepare() error = %v", err)
	}
	return prepared
}

func testProject(tb testing.TB) string {
	tb.Helper()
	workDir := tb.TempDir()
	cfg := config.DefaultConfig()
	if err := cfg.Save(filepath.Join(workDir, ".terraci.yaml")); err != nil {
		tb.Fatalf("Save config: %v", err)
	}
	moduleDir := filepath.Join(workDir, "platform", "stage", "eu-central-1", "vpc")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		tb.Fatalf("mkdir module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.tf"), []byte("# test\n"), 0o644); err != nil {
		tb.Fatalf("write module: %v", err)
	}
	return workDir
}
