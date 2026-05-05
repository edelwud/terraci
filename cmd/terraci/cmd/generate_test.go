package cmd

import (
	"context"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

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
	string,
	string,
	*discovery.ModuleIndex,
) ([]*discovery.Module, []string, error) {
	return nil, nil, nil
}

func (d generateTargetTestChangeDetector) DetectChangedLibraries(
	context.Context,
	string,
	string,
	[]string,
) ([]string, error) {
	return d.changedLibraries, nil
}

func TestResolveGenerateTargetsUsesWorkflowResolveTargets(t *testing.T) {
	plugins := registry.NewFromFactories(func() plugin.Plugin {
		return generateTargetTestChangeDetector{changedLibraries: []string{"_modules/network"}}
	})

	stage := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	prod := discovery.TestModule("platform", "prod", "eu-central-1", "vpc")
	depGraph := graph.BuildFromDependencies([]*discovery.Module{stage, prod}, nil)
	depGraph.AddLibraryUsage("_modules/network", stage.ID())
	depGraph.AddLibraryUsage("_modules/network", prod.ID())

	result := &workflow.Result{
		All:      workflow.NewModuleSet([]*discovery.Module{stage, prod}),
		Filtered: workflow.NewModuleSet([]*discovery.Module{stage}),
		Graph:    depGraph,
	}
	cfg := config.DefaultConfig()
	cfg.LibraryModules = &config.LibraryModulesConfig{Paths: []string{"_modules"}}
	app := &App{
		Config:  cfg,
		WorkDir: t.TempDir(),
		Version: "test",
		Plugins: plugins,
	}
	ff := &filter.Flags{SegmentArgs: []string{"environment=stage"}}

	cmd := &cobra.Command{}
	appCtx := plugin.NewAppContext(plugin.AppContextOptions{
		Config:   app.Config,
		WorkDir:  app.WorkDir,
		Version:  app.Version,
		Resolver: plugins,
	})
	cmd.SetContext(plugin.WithContext(context.Background(), appCtx))

	got, err := resolveGenerateTargets(cmd, app, result, true, "main", ff)
	if err != nil {
		t.Fatalf("resolveGenerateTargets() error = %v", err)
	}
	want, err := workflow.ResolveTargets(context.Background(), app.WorkDir, app.Config, result, workflow.TargetSelectionOptions{
		ChangedOnly: true,
		BaseRef:     "main",
		Filters:     ff,
		ChangeDetectorResolver: func() (workflow.ChangeDetector, error) {
			return app.Plugins.ResolveChangeDetector()
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

func targetIDs(modules []*discovery.Module) []string {
	ids := make([]string, len(modules))
	for i := range modules {
		ids[i] = modules[i].ID()
	}
	return ids
}
