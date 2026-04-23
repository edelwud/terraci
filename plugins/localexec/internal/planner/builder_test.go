package planner

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

type stubContributionCollector struct {
	contributions []*pipeline.Contribution
	calls         int
}

func (c *stubContributionCollector) Collect(*plugin.AppContext) []*pipeline.Contribution {
	c.calls++
	return c.contributions
}

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

func TestBuilderBuildUsesInjectedContributionCollector(t *testing.T) {
	t.Parallel()

	appCtx := plugintest.NewAppContext(t, t.TempDir())
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)
	collector := &stubContributionCollector{
		contributions: []*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{{
				Name:     "summary",
				Phase:    pipeline.PhaseFinalize,
				Commands: []string{"terraci summary"},
			}},
		}},
	}

	plan, err := NewWithContributionCollector(appCtx, collector).Build(
		[]*discovery.Module{module},
		result,
		execution.Config{PlanEnabled: true},
		spec.ExecutionModePlan,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if collector.calls != 1 {
		t.Fatalf("collector calls = %d, want 1", collector.calls)
	}
	if jobs := plan.JobsByPhase(pipeline.PhaseFinalize); len(jobs) != 1 || jobs[0].Name != "summary" {
		t.Fatalf("finalize jobs = %#v, want summary job", jobs)
	}
}

func TestBuilderBuildPlanModeExcludesApplyPhaseContributedJobs(t *testing.T) {
	t.Parallel()

	appCtx := plugintest.NewAppContext(t, t.TempDir())
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)
	collector := &stubContributionCollector{
		contributions: []*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{
				{Name: "pre-plan", Phase: pipeline.PhasePrePlan, Commands: []string{"pre-plan"}},
				{Name: "post-plan", Phase: pipeline.PhasePostPlan, Commands: []string{"post-plan"}},
				{Name: "pre-apply", Phase: pipeline.PhasePreApply, Commands: []string{"pre-apply"}},
				{Name: "post-apply", Phase: pipeline.PhasePostApply, Commands: []string{"post-apply"}},
				{Name: "summary", Phase: pipeline.PhaseFinalize, Commands: []string{"summary"}},
			},
		}},
	}

	plan, err := NewWithContributionCollector(appCtx, collector).Build(
		[]*discovery.Module{module},
		result,
		execution.Config{PlanEnabled: true},
		spec.ExecutionModePlan,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if jobs := plan.JobsByPhase(pipeline.PhasePrePlan); len(jobs) != 1 || jobs[0].Name != "pre-plan" {
		t.Fatalf("pre-plan jobs = %#v, want pre-plan job", jobs)
	}
	if jobs := plan.JobsByPhase(pipeline.PhasePostPlan); len(jobs) != 1 || jobs[0].Name != "post-plan" {
		t.Fatalf("post-plan jobs = %#v, want post-plan job", jobs)
	}
	if jobs := plan.JobsByPhase(pipeline.PhasePreApply); len(jobs) != 0 {
		t.Fatalf("pre-apply jobs = %#v, want none", jobs)
	}
	if jobs := plan.JobsByPhase(pipeline.PhasePostApply); len(jobs) != 0 {
		t.Fatalf("post-apply jobs = %#v, want none", jobs)
	}
	if jobs := plan.JobsByPhase(pipeline.PhaseFinalize); len(jobs) != 1 || jobs[0].Name != "summary" {
		t.Fatalf("finalize jobs = %#v, want summary job", jobs)
	}
}

func TestBuilderBuildRunModeKeepsApplyPhaseContributedJobs(t *testing.T) {
	t.Parallel()

	appCtx := plugintest.NewAppContext(t, t.TempDir())
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)
	collector := &stubContributionCollector{
		contributions: []*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{{
				Name:     "pre-apply",
				Phase:    pipeline.PhasePreApply,
				Commands: []string{"pre-apply"},
			}},
		}},
	}

	plan, err := NewWithContributionCollector(appCtx, collector).Build(
		[]*discovery.Module{module},
		result,
		execution.Config{PlanEnabled: true},
		spec.ExecutionModeRun,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if jobs := plan.JobsByPhase(pipeline.PhasePreApply); len(jobs) != 1 || jobs[0].Name != "pre-apply" {
		t.Fatalf("pre-apply jobs = %#v, want pre-apply job", jobs)
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
