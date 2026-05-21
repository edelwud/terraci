package projectflow

import (
	"context"
	"reflect"
	"testing"

	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/pkg/workflow"
)

type testChangeDetector struct {
	changedLibraries []string
}

func (d testChangeDetector) Name() string { return "projectflow-test" }

func (d testChangeDetector) Description() string {
	return "projectflow test change detector"
}

func (d testChangeDetector) DetectChanges(context.Context, workflow.ChangeDetectionRequest) (*workflow.ChangeDetectionResult, error) {
	return &workflow.ChangeDetectionResult{LibraryPaths: d.changedLibraries}, nil
}

func TestResolveTargetsUsesWorkflowResolveTargets(t *testing.T) {
	plugins := registry.NewFromFactories(func() plugin.Plugin {
		return testChangeDetector{changedLibraries: []string{"_modules/network"}}
	})

	stage := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	prod := discovery.TestModule("platform", "prod", "eu-central-1", "vpc")
	depGraph := graph.BuildFromDependencies([]*discovery.Module{stage, prod}, nil)
	depGraph.AddLibraryUsage("_modules/network", stage.ID())
	depGraph.AddLibraryUsage("_modules/network", prod.ID())

	workflowResult := &workflow.Result{
		All:      workflow.NewModuleSet([]*discovery.Module{stage, prod}),
		Filtered: workflow.NewModuleSet([]*discovery.Module{stage}),
		Graph:    depGraph,
	}
	cfg := config.DefaultConfig()
	cfg.LibraryModules = &config.LibraryModulesConfig{Paths: []string{"_modules"}}
	workDir := t.TempDir()
	if err := cfg.Save(workDir + "/.terraci.yaml"); err != nil {
		t.Fatalf("Save config: %v", err)
	}
	flags := filter.Flags{SegmentArgs: []string{"environment=stage"}}

	prepared, err := runflow.New(runflow.Options{
		RegistryFactory: func() *registry.Registry { return plugins },
	}).Prepare(context.Background(), runflow.Request{
		CommandName: "projectflow-test",
		WorkDir:     workDir,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	got, err := ResolveTargets(context.Background(), NewRuntime(prepared), workflowResult, Request{
		Filters:     flags,
		ChangedOnly: true,
		BaseRef:     "main",
	})
	if err != nil {
		t.Fatalf("ResolveTargets() error = %v", err)
	}
	want, err := workflow.ResolveTargets(context.Background(), workDir, prepared.Config(), workflowResult, workflow.TargetSelectionOptions{
		ChangedOnly: true,
		BaseRef:     "main",
		Filters:     &flags,
		ChangeDetectorResolver: func() (workflow.ChangeDetector, error) {
			return plugins.ResolveChangeDetector()
		},
	})
	if err != nil {
		t.Fatalf("workflow.ResolveTargets() error = %v", err)
	}

	if !reflect.DeepEqual(targetIDs(got), targetIDs(want)) {
		t.Fatalf("target ids = %v, want %v", targetIDs(got), targetIDs(want))
	}
	if !reflect.DeepEqual(targetIDs(got), []string{stage.ID()}) {
		t.Fatalf("target ids = %v, want [%s]", targetIDs(got), stage.ID())
	}
}

func TestSummarizeLibraries(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LibraryModules = &config.LibraryModulesConfig{Paths: []string{"_modules"}}

	used := discovery.TestLibraryModule("_modules/kafka", "/abs/_modules/kafka")
	orphan := discovery.TestLibraryModule("_modules/unused", "/abs/_modules/unused")
	nestedConsumed := discovery.TestLibraryModule("_modules/kafka_acl", "/abs/_modules/kafka_acl")

	depGraph := graph.NewDependencyGraph()
	depGraph.AddLibraryUsage("/abs/_modules/kafka", "platform/stage/eu-central-1/msk")
	depGraph.AddLibraryUsage("/abs/_modules/kafka_acl/sub", "platform/stage/eu-central-1/msk")

	workflowResult := &workflow.Result{
		Libraries: workflow.NewModuleSet([]*discovery.Module{used, orphan, nestedConsumed}),
		Graph:     depGraph,
	}

	summary := SummarizeLibraries(cfg.Snapshot(), workflowResult)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if summary.ConfiguredPaths != 1 {
		t.Errorf("ConfiguredPaths = %d, want 1", summary.ConfiguredPaths)
	}
	if summary.Discovered != 3 {
		t.Errorf("Discovered = %d, want 3", summary.Discovered)
	}
	if summary.Consumers != 1 {
		t.Errorf("Consumers = %d, want 1", summary.Consumers)
	}
	if !reflect.DeepEqual(summary.Orphans, []string{"_modules/unused"}) {
		t.Errorf("Orphans = %v, want [_modules/unused]", summary.Orphans)
	}
}

func TestSummarizeLibrariesNoConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	workflowResult := &workflow.Result{Graph: graph.NewDependencyGraph()}

	if got := SummarizeLibraries(cfg.Snapshot(), workflowResult); got != nil {
		t.Errorf("expected nil summary when library_modules is unset, got %+v", got)
	}
}

func targetIDs(modules []*discovery.Module) []string {
	ids := make([]string, len(modules))
	for i := range modules {
		ids[i] = modules[i].ID()
	}
	return ids
}
