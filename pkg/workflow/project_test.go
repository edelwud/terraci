package workflow

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
)

type projectTestChangeDetector struct {
	modules          []*discovery.Module
	changedLibraries []string
	baseRef          string
	libraryPaths     []string
}

func (d *projectTestChangeDetector) DetectChanges(_ context.Context, req ChangeDetectionRequest) (*ChangeDetectionResult, error) {
	d.baseRef = req.BaseRef
	d.libraryPaths = append([]string(nil), req.LibraryPaths...)
	return &ChangeDetectionResult{Modules: d.modules, LibraryPaths: d.changedLibraries}, nil
}

func TestPlanProjectNoTargetMode(t *testing.T) {
	workDir := testProjectDir(t, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/prod/eu-central-1/vpc",
	})

	result, err := PlanProject(context.Background(), ProjectRequest{WorkDir: workDir})
	if err != nil {
		t.Fatalf("PlanProject() error = %v", err)
	}
	if result.Workflow == nil {
		t.Fatal("Workflow = nil")
	}
	if len(result.Workflow.Filtered.Modules) != 2 {
		t.Fatalf("filtered modules = %d, want 2", len(result.Workflow.Filtered.Modules))
	}
	if result.Targets != nil {
		t.Fatalf("Targets = %#v, want nil without targeting", result.Targets)
	}
}

func TestPlanProjectTargetMode(t *testing.T) {
	workDir := testProjectDir(t, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/prod/eu-central-1/vpc",
	})

	result, err := PlanProject(context.Background(), ProjectRequest{
		WorkDir: workDir,
		Filters: filter.Flags{
			SegmentArgs: []string{"environment=stage"},
		},
		Targeting: TargetRequest{Enabled: true},
	})
	if err != nil {
		t.Fatalf("PlanProject() error = %v", err)
	}
	if got := moduleIDs(result.Targets); !reflect.DeepEqual(got, []string{"platform/stage/eu-central-1/vpc"}) {
		t.Fatalf("target ids = %v, want stage vpc", got)
	}
}

func TestPlanProjectChangedOnlyPropagatesBaseRefAndLibraryRoots(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LibraryModules = &config.LibraryModulesConfig{Paths: []string{"_modules"}}
	workDir := testProjectDirWithConfig(t, cfg, []string{
		"platform/stage/eu-central-1/vpc",
		"platform/prod/eu-central-1/vpc",
		"_modules/network",
	})
	stage := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	detector := &projectTestChangeDetector{
		modules:          []*discovery.Module{stage},
		changedLibraries: []string{"_modules/network"},
	}

	result, err := PlanProject(context.Background(), ProjectRequest{
		WorkDir: workDir,
		Config:  cfg.Snapshot(),
		Filters: filter.Flags{
			SegmentArgs: []string{"environment=stage"},
		},
		Targeting: TargetRequest{
			Enabled:                true,
			ChangedOnly:            true,
			BaseRef:                "origin/main",
			ChangeDetectorResolver: func() (ChangeDetector, error) { return detector, nil },
		},
	})
	if err != nil {
		t.Fatalf("PlanProject() error = %v", err)
	}
	if detector.baseRef != "origin/main" {
		t.Fatalf("base ref = %q, want origin/main", detector.baseRef)
	}
	if !reflect.DeepEqual(detector.libraryPaths, []string{"_modules"}) {
		t.Fatalf("library roots = %v, want [_modules]", detector.libraryPaths)
	}
	if got := moduleIDs(result.Targets); !reflect.DeepEqual(got, []string{"platform/stage/eu-central-1/vpc"}) {
		t.Fatalf("target ids = %v, want stage vpc", got)
	}
}

func TestPlanProjectChangedOnlyNoTargets(t *testing.T) {
	workDir := testProjectDir(t, []string{"platform/stage/eu-central-1/vpc"})
	detector := &projectTestChangeDetector{}

	result, err := PlanProject(context.Background(), ProjectRequest{
		WorkDir: workDir,
		Targeting: TargetRequest{
			Enabled:                true,
			ChangedOnly:            true,
			ChangeDetectorResolver: func() (ChangeDetector, error) { return detector, nil },
		},
	})
	if err != nil {
		t.Fatalf("PlanProject() error = %v", err)
	}
	if len(result.Targets) != 0 {
		t.Fatalf("targets = %v, want none", moduleIDs(result.Targets))
	}
}

func TestProjectLibraryDiagnostics(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LibraryModules = &config.LibraryModulesConfig{Paths: []string{"_modules"}}

	used := discovery.TestLibraryModule("_modules/kafka", "/abs/_modules/kafka")
	orphan := discovery.TestLibraryModule("_modules/unused", "/abs/_modules/unused")
	nestedConsumed := discovery.TestLibraryModule("_modules/kafka_acl", "/abs/_modules/kafka_acl")

	depGraph := graph.NewDependencyGraph()
	depGraph.AddLibraryUsage("/abs/_modules/kafka", "platform/stage/eu-central-1/msk")
	depGraph.AddLibraryUsage("/abs/_modules/kafka_acl/sub", "platform/stage/eu-central-1/msk")

	workflowResult := &Result{
		Libraries: NewModuleSet([]*discovery.Module{used, orphan, nestedConsumed}),
		Graph:     depGraph,
	}

	summary := SummarizeLibraries(cfg.Snapshot(), workflowResult)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if summary.ConfiguredPaths != 1 || summary.Discovered != 3 || summary.Consumers != 1 {
		t.Fatalf("summary = %+v, want paths=1 discovered=3 consumers=1", summary)
	}
	if !reflect.DeepEqual(summary.Orphans, []string{"_modules/unused"}) {
		t.Fatalf("orphans = %v, want [_modules/unused]", summary.Orphans)
	}
}

func testProjectDir(tb testing.TB, modules []string) string {
	tb.Helper()
	return testProjectDirWithConfig(tb, config.DefaultConfig(), modules)
}

func testProjectDirWithConfig(tb testing.TB, cfg *config.Config, modules []string) string {
	tb.Helper()
	workDir := tb.TempDir()
	if err := cfg.Save(filepath.Join(workDir, ".terraci.yaml")); err != nil {
		tb.Fatalf("Save config: %v", err)
	}
	for _, module := range modules {
		dir := filepath.Join(workDir, module)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tb.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte("# test\n"), 0o644); err != nil {
			tb.Fatalf("write %s/main.tf: %v", dir, err)
		}
	}
	return workDir
}
