package cmd

import (
	"context"
	"reflect"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/pkg/workflow"
)

type generateTargetTestChangeDetector struct {
	changedLibraries []string
}

func (d generateTargetTestChangeDetector) Name() string { return "generate-target-test" }

func (d generateTargetTestChangeDetector) Description() string {
	return "generate target test change detector"
}

func (d generateTargetTestChangeDetector) DetectChangedModules(
	context.Context,
	*plugin.AppContext,
	string,
	*discovery.ModuleIndex,
) ([]*discovery.Module, []string, error) {
	return nil, nil, nil
}

func (d generateTargetTestChangeDetector) DetectChangedLibraries(
	context.Context,
	*plugin.AppContext,
	string,
	[]string,
) ([]string, error) {
	return d.changedLibraries, nil
}

func TestResolveGenerateTargetsUsesWorkflowResolveTargets(t *testing.T) {
	registry.Reset()
	t.Cleanup(registry.Reset)
	registry.Register(generateTargetTestChangeDetector{changedLibraries: []string{"_modules/network"}})

	stage := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	prod := discovery.TestModule("platform", "prod", "eu-central-1", "vpc")
	depGraph := graph.BuildFromDependencies([]*discovery.Module{stage, prod}, nil)
	depGraph.AddLibraryUsage("_modules/network", stage.ID())
	depGraph.AddLibraryUsage("_modules/network", prod.ID())

	result := &workflow.Result{
		FilteredModules: []*discovery.Module{stage},
		FullIndex:       discovery.NewModuleIndex([]*discovery.Module{stage, prod}),
		FilteredIndex:   discovery.NewModuleIndex([]*discovery.Module{stage}),
		Graph:           depGraph,
	}
	cfg := config.DefaultConfig()
	cfg.LibraryModules = &config.LibraryModulesConfig{Paths: []string{"_modules"}}
	app := &App{
		Config:  cfg,
		WorkDir: t.TempDir(),
		Version: "test",
	}
	ff := &filter.Flags{SegmentArgs: []string{"environment=stage"}}

	got, err := resolveGenerateTargets(context.Background(), app, result, true, "main", ff)
	if err != nil {
		t.Fatalf("resolveGenerateTargets() error = %v", err)
	}
	want, err := workflow.ResolveTargets(context.Background(), app.PluginContext(), result, workflow.TargetSelectionOptions{
		ChangedOnly:            true,
		BaseRef:                "main",
		Filters:                ff,
		ChangeDetectorResolver: registry.ResolveChangeDetector,
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

func targetIDs(modules []*discovery.Module) []string {
	ids := make([]string, len(modules))
	for i := range modules {
		ids[i] = modules[i].ID()
	}
	return ids
}
