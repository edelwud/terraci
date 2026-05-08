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

	if got := len(plan.PlanJobsForLevel(0)); got != 1 {
		t.Fatalf("plan jobs = %d, want 1", got)
	}
	if got := len(plan.ApplyJobsForLevel(0)); got != 1 {
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

	if got := len(plan.PlanJobsForLevel(0)); got != 1 {
		t.Fatalf("plan jobs = %d, want 1", got)
	}
	if got := len(plan.ApplyJobsForLevel(0)); got != 0 {
		t.Fatalf("apply jobs = %d, want 0", got)
	}
}

func TestBuilderBuildUsesContributionSnapshot(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)
	contributions := []*pipeline.Contribution{{
		Jobs: []pipeline.ContributedJob{{
			Name:     "summary",
			Commands: []string{"terraci summary"},
		}},
	}}

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

	if len(plan.Jobs) != 1 || plan.Jobs[0].Name != "summary" {
		t.Fatalf("contributed jobs = %#v, want summary job", plan.Jobs)
	}
}

func TestBuilderBuildPlanModeKeepsAllContributedJobs(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	result := workflowResultForModules(module)
	contributions := []*pipeline.Contribution{{
		Jobs: []pipeline.ContributedJob{
			{Name: "lint", Commands: []string{"terraci lint"}},
			{Name: "cost", Commands: []string{"terraci cost"}},
			{Name: "policy", Commands: []string{"terraci policy check"}},
			{Name: "tfupdate", Commands: []string{"terraci tfupdate"}},
			{Name: "summary", Commands: []string{"summary"}},
		},
	}}

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

	if len(plan.Jobs) != 5 {
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

	planJob := plan.Levels[0].Modules[0].Plan
	if !planJob.Operation.Terraform.DetailedPlan {
		t.Fatal("detailed plan mode should request detailed plan resources")
	}
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
