package planner

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

func mustContribution(tb testing.TB, opts ...pipeline.ContributedJobOptions) *pipeline.Contribution {
	tb.Helper()
	jobs := make([]pipeline.ContributedJob, 0, len(opts))
	for _, opt := range opts {
		job, err := pipeline.NewContributedJob(opt)
		if err != nil {
			tb.Fatalf("NewContributedJob() error = %v", err)
		}
		jobs = append(jobs, job)
	}
	contribution, err := pipeline.NewContribution(jobs...)
	if err != nil {
		tb.Fatalf("NewContribution() error = %v", err)
	}
	return contribution
}

func TestBuilderBuildRunModeIncludesPlanAndApplyJobs(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)

	plan, err := New().Build(
		[]*discovery.Module{module},
		result,
		execution.Config{InitEnabled: true, PlanEnabled: true},
		spec.ExecutionModeRun,
		nil,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got := countJobsByKind(plan, pipeline.JobKindPlan); got != 1 {
		t.Fatalf("plan jobs = %d, want 1", got)
	}
	if got := countJobsByKind(plan, pipeline.JobKindApply); got != 1 {
		t.Fatalf("apply jobs = %d, want 1", got)
	}
}

func TestBuilderBuildPlanModeForcesPlanOnly(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)

	plan, err := New().Build(
		[]*discovery.Module{module},
		result,
		execution.Config{PlanEnabled: false},
		spec.ExecutionModePlan,
		nil,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got := countJobsByKind(plan, pipeline.JobKindPlan); got != 1 {
		t.Fatalf("plan jobs = %d, want 1", got)
	}
	if got := countJobsByKind(plan, pipeline.JobKindApply); got != 0 {
		t.Fatalf("apply jobs = %d, want 0", got)
	}
}

func TestBuilderBuildUsesContributionSnapshot(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)
	contributions := []*pipeline.Contribution{mustContribution(t, pipeline.ContributedJobOptions{
		Name:     "summary",
		Commands: []string{"terraci summary"},
	})}

	plan, err := New().Build(
		[]*discovery.Module{module},
		result,
		execution.Config{PlanEnabled: true},
		spec.ExecutionModePlan,
		contributions,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got := countJobsByKind(plan, pipeline.JobKindCommand); got != 1 || findJob(plan, "summary") == nil {
		t.Fatalf("contributed jobs = %#v, want summary job", plan.Jobs)
	}
}

func TestBuilderBuildPlanModeKeepsAllContributedJobs(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)
	contributions := []*pipeline.Contribution{mustContribution(t,
		pipeline.ContributedJobOptions{Name: "lint", Commands: []string{"terraci lint"}},
		pipeline.ContributedJobOptions{Name: "cost", Commands: []string{"terraci cost"}},
		pipeline.ContributedJobOptions{Name: "policy", Commands: []string{"terraci policy check"}},
		pipeline.ContributedJobOptions{Name: "tfupdate", Commands: []string{"terraci tfupdate"}},
		pipeline.ContributedJobOptions{Name: "summary", Commands: []string{"summary"}},
	)}

	plan, err := New().Build(
		[]*discovery.Module{module},
		result,
		execution.Config{PlanEnabled: true},
		spec.ExecutionModePlan,
		contributions,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if got := countJobsByKind(plan, pipeline.JobKindCommand); got != 5 {
		t.Fatalf("contributed jobs = %#v, want all jobs in plan mode", plan.Jobs)
	}
}

func TestBuilderBuildDetailedPlanModeRequestsDetailedResources(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)

	plan, err := New().Build(
		[]*discovery.Module{module},
		result,
		execution.Config{PlanEnabled: true, PlanMode: execution.PlanModeDetailed},
		spec.ExecutionModeRun,
		nil,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	planJob := findJob(plan, pipeline.JobName(pipeline.JobKindPlan, module))
	if planJob == nil {
		t.Fatal("plan job not found")
	}
	if !planJob.Operation.Terraform.DetailedPlan {
		t.Fatal("detailed plan mode should request detailed plan resources")
	}
}

func countJobsByKind(ir *pipeline.IR, kind pipeline.JobKind) int {
	count := 0
	for i := range ir.Jobs {
		if ir.Jobs[i].Kind == kind {
			count++
		}
	}
	return count
}

func findJob(ir *pipeline.IR, name string) *pipeline.Job {
	for i := range ir.Jobs {
		if ir.Jobs[i].Name == name {
			return &ir.Jobs[i]
		}
	}
	return nil
}

func workflowResultForModules(modules ...*discovery.Module) *workflow.Result {
	depGraph := graph.NewDependencyGraph()
	for _, module := range modules {
		depGraph.AddNode(module)
	}

	return &workflow.Result{
		Filtered: workflow.NewModuleSet(modules),
		Graph:    depGraph,
	}
}
