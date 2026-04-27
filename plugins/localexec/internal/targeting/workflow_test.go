package targeting

import (
	"context"
	"reflect"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

type workflowResolverTestChangeDetector struct {
	changedLibraries []string
}

func (d workflowResolverTestChangeDetector) Name() string { return "localexec-target-test" }

func (d workflowResolverTestChangeDetector) Description() string {
	return "localexec target test change detector"
}

func (d workflowResolverTestChangeDetector) DetectChangedModules(
	context.Context,
	*plugin.AppContext,
	string,
	*discovery.ModuleIndex,
) ([]*discovery.Module, []string, error) {
	return nil, nil, nil
}

func (d workflowResolverTestChangeDetector) DetectChangedLibraries(
	context.Context,
	*plugin.AppContext,
	string,
	[]string,
) ([]string, error) {
	return d.changedLibraries, nil
}

func TestWorkflowResolverUsesWorkflowResolveTargets(t *testing.T) {
	plugins := registry.NewFromFactories(func() plugin.Plugin {
		return workflowResolverTestChangeDetector{changedLibraries: []string{"_modules/network"}}
	})

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
	appCtx := plugintest.NewAppContext(t, t.TempDir())
	cfg := appCtx.Config()
	cfg.LibraryModules = &config.LibraryModulesConfig{Paths: []string{"_modules"}}
	appCtx = plugin.NewAppContext(cfg, appCtx.WorkDir(), appCtx.ServiceDir(), appCtx.Version(), appCtx.Reports(), plugins)
	filters := &filter.Flags{SegmentArgs: []string{"environment=stage"}}
	req := spec.ExecuteRequest{
		ChangedOnly: true,
		BaseRef:     "main",
		ModulePath:  stage.RelativePath,
		Filters:     filters,
	}

	got, err := NewWorkflowResolver(appCtx, plugins.ResolveChangeDetector).Resolve(context.Background(), req, result)
	if err != nil {
		t.Fatalf("WorkflowResolver.Resolve() error = %v", err)
	}
	want, err := workflow.ResolveTargets(context.Background(), appCtx, result, workflow.TargetSelectionOptions{
		ModulePath:             req.ModulePath,
		ChangedOnly:            req.ChangedOnly,
		BaseRef:                req.BaseRef,
		Filters:                req.Filters,
		ChangeDetectorResolver: plugins.ResolveChangeDetector,
	})
	if err != nil {
		t.Fatalf("workflow.ResolveTargets() error = %v", err)
	}

	if !reflect.DeepEqual(workflowResolverTargetIDs(got), workflowResolverTargetIDs(want)) {
		t.Fatalf("target ids = %v, want %v", workflowResolverTargetIDs(got), workflowResolverTargetIDs(want))
	}
	if !reflect.DeepEqual(workflowResolverTargetIDs(got), []string{stage.ID()}) {
		t.Fatalf("target ids = %v, want [%s]", workflowResolverTargetIDs(got), stage.ID())
	}
}

func workflowResolverTargetIDs(modules []*discovery.Module) []string {
	ids := make([]string, len(modules))
	for i := range modules {
		ids[i] = modules[i].ID()
	}
	return ids
}
