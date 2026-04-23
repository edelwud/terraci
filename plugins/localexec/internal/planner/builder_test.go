package planner

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

func TestBuilderBuildRunModeIncludesPlanAndApplyJobs(t *testing.T) {
	t.Parallel()

	appCtx := plugintest.NewAppContext(t, t.TempDir())
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)

	plan, err := New(appCtx).Build(
		[]*discovery.Module{module},
		result,
		execution.Config{InitEnabled: true, PlanEnabled: true},
		spec.ExecutionModeRun,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got := len(plan.PlanJobsForLevel(0)); got != 1 {
		t.Fatalf("plan jobs = %d, want 1", got)
	}
	if got := len(plan.ApplyJobsForLevel(0)); got != 1 {
		t.Fatalf("apply jobs = %d, want 1", got)
	}
}

func TestBuilderBuildPlanModeForcesPlanOnly(t *testing.T) {
	t.Parallel()

	appCtx := plugintest.NewAppContext(t, t.TempDir())
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)

	plan, err := New(appCtx).Build(
		[]*discovery.Module{module},
		result,
		execution.Config{PlanEnabled: false},
		spec.ExecutionModePlan,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got := len(plan.PlanJobsForLevel(0)); got != 1 {
		t.Fatalf("plan jobs = %d, want 1", got)
	}
	if got := len(plan.ApplyJobsForLevel(0)); got != 0 {
		t.Fatalf("apply jobs = %d, want 0", got)
	}
}

func workflowResultForModules(modules ...*discovery.Module) *workflow.Result {
	depGraph := graph.NewDependencyGraph()
	for _, module := range modules {
		depGraph.AddNode(module)
	}

	return &workflow.Result{
		FilteredModules: modules,
		FilteredIndex:   discovery.NewModuleIndex(modules),
		Graph:           depGraph,
	}
}
