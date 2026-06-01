package test

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/workflow"
)

func mustPipelineContribution(tb testing.TB, opts ...pipeline.ContributedJobOptions) *pipeline.Contribution {
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

func TestPipelineBuild_BasicModules(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "prod", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "prod", "eu-central-1", "eks"),
	}

	// eks depends on vpc
	deps := map[string]*parser.ModuleDependencies{
		modules[0].ID(): {},
		modules[1].ID(): {DependsOn: []string{modules[0].ID()}},
	}

	depGraph := graph.BuildFromDependencies(modules, deps)
	ir, err := buildPipelineIR(modules, depGraph, modules, pipeline.ProjectIRRequest{
		Script: pipeline.ScriptConfig{
			InitEnabled: true,
		},
		Intent: mustBuildIntent(t, true),
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if got := ir.ModuleCount(); got != 2 {
		t.Errorf("expected 2 modules, got %d", got)
	}

	for _, mod := range modules {
		if _, ok := ir.JobForModule(pipeline.JobKindPlan, mod); !ok {
			t.Errorf("module %s missing plan job", mod.ID())
		}
		if _, ok := ir.JobForModule(pipeline.JobKindApply, mod); !ok {
			t.Errorf("module %s missing apply job", mod.ID())
		}
	}
}

func TestPipelineBuild_PlanOnly(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "prod", "eu", "app"),
	}
	deps := map[string]*parser.ModuleDependencies{
		modules[0].ID(): {},
	}
	depGraph := graph.BuildFromDependencies(modules, deps)
	ir, err := buildPipelineIR(modules, depGraph, modules, pipeline.ProjectIRRequest{
		Intent: mustBuildIntent(t, false, pipeline.AllPlanResources(pipeline.ResourceKindPlanBinary)),
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if _, ok := ir.JobForModule(pipeline.JobKindPlan, modules[0]); !ok {
		t.Error("plan job should exist in plan-only mode")
	}
	if _, ok := ir.JobForModule(pipeline.JobKindApply, modules[0]); ok {
		t.Error("apply job should not exist in plan-only mode")
	}
}

func TestPipelineBuild_WithContributions(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "prod", "eu", "app"),
	}
	deps := map[string]*parser.ModuleDependencies{
		modules[0].ID(): {},
	}
	depGraph := graph.BuildFromDependencies(modules, deps)
	contributions := []*pipeline.Contribution{mustPipelineContribution(t, pipeline.ContributedJobOptions{
		Name:     "policy-check",
		Commands: []string{"terraci policy check --format text"},
		Consumes: []pipeline.ResourceRequest{
			pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
		},
	})}

	ir, err := buildPipelineIR(modules, depGraph, modules, pipeline.ProjectIRRequest{
		Intent:        mustBuildIntent(t, true),
		Contributions: contributions,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	policyJob, ok := ir.FindJob("policy-check")
	if !ok {
		t.Fatal("expected policy-check job")
	}

	// policy-check should depend on the plan job
	planJob, ok := ir.JobForModule(pipeline.JobKindPlan, modules[0])
	if !ok {
		t.Fatal("missing plan job")
	}
	planName := planJob.Name()
	if !policyJob.DependsOn(planJob) {
		t.Error("policy-check should depend on the plan job")
	}
	if !hasPipelineInputArtifact(policyJob.InputArtifacts(), pipeline.PlanArtifactName(planName), planName) {
		t.Error("policy-check should restore the plan artifact")
	}
}

func TestPipelineBuild_SummaryDependsThroughResources(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "prod", "eu", "app"),
	}
	deps := map[string]*parser.ModuleDependencies{modules[0].ID(): {}}
	depGraph := graph.BuildFromDependencies(modules, deps)

	contributions := []*pipeline.Contribution{
		mustPipelineContribution(t, pipeline.ContributedJobOptions{
			Name:     "policy-check",
			Commands: []string{"check"},
			Produces: []pipeline.ResourceSpec{
				pipeline.PluginResource(pipeline.ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
			},
		}),
		mustPipelineContribution(t, pipeline.ContributedJobOptions{
			Name:     "terraci-summary",
			Commands: []string{"summary"},
			Consumes: []pipeline.ResourceRequest{
				pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
				pipeline.AllPluginResources(pipeline.ResourceKindPluginReport, true),
			},
		}),
	}

	ir, err := buildPipelineIR(modules, depGraph, modules, pipeline.ProjectIRRequest{
		Intent:        mustBuildIntent(t, true),
		Contributions: contributions,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	summaryJob, ok := ir.FindJob("terraci-summary")
	if !ok {
		t.Fatal("missing summary job")
	}

	planJob, ok := ir.JobForModule(pipeline.JobKindPlan, modules[0])
	if !ok {
		t.Fatal("missing plan job")
	}
	planName := planJob.Name()
	if !summaryJob.DependsOnName("policy-check") {
		t.Error("summary job should depend on policy-check report")
	}
	if !summaryJob.DependsOn(planJob) {
		t.Error("summary job should depend on plan jobs")
	}
	if !hasPipelineInputArtifact(summaryJob.InputArtifacts(), pipeline.ResultArtifactName("policy-check"), "policy-check") {
		t.Error("summary job should restore policy report artifact")
	}
	if !hasPipelineInputArtifact(summaryJob.InputArtifacts(), pipeline.PlanArtifactName(planName), planName) {
		t.Error("summary job should restore plan artifact")
	}
}

func TestPipelineBuild_DependencyOrdering(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "prod", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "prod", "eu-central-1", "eks"),
		discovery.TestModule("platform", "prod", "eu-central-1", "app"),
	}

	// app -> eks -> vpc
	deps := map[string]*parser.ModuleDependencies{
		modules[0].ID(): {},
		modules[1].ID(): {DependsOn: []string{modules[0].ID()}},
		modules[2].ID(): {DependsOn: []string{modules[1].ID()}},
	}

	depGraph := graph.BuildFromDependencies(modules, deps)
	ir, err := buildPipelineIR(modules, depGraph, modules, pipeline.ProjectIRRequest{
		Intent: mustBuildIntent(t, true),
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	groups, err := pipeline.Schedule(ir)
	if err != nil {
		t.Fatalf("Schedule failed: %v", err)
	}
	if len(groups) < 2 {
		t.Errorf("expected multiple DAG groups for a dependency chain, got %d", len(groups))
	}

	eksPlan, ok := ir.JobForModule(pipeline.JobKindPlan, modules[1])
	if !ok {
		t.Fatal("missing eks plan job")
	}
	vpcApply, ok := ir.JobForModule(pipeline.JobKindApply, modules[0])
	if !ok {
		t.Fatal("missing vpc apply job")
	}
	if !eksPlan.DependsOn(vpcApply) {
		t.Error("dependent module plan should depend on upstream apply in full run mode")
	}
}

func buildPipelineIR(allModules []*discovery.Module, depGraph *graph.DependencyGraph, targets []*discovery.Module, req pipeline.ProjectIRRequest) (*pipeline.IR, error) {
	req.Project = &workflow.ProjectResult{
		Workflow: &workflow.Result{
			Filtered: workflow.NewModuleSet(allModules),
			Graph:    depGraph,
		},
		Targets: targets,
	}
	return pipeline.BuildProjectIR(req)
}

func mustBuildIntent(tb testing.TB, applyEnabled bool, requests ...pipeline.ResourceRequest) pipeline.BuildIntent {
	tb.Helper()
	intent, err := pipeline.NewBuildIntent(pipeline.BuildIntentOptions{
		ApplyEnabled:     applyEnabled,
		ResourceRequests: requests,
	})
	if err != nil {
		tb.Fatalf("NewBuildIntent() error = %v", err)
	}
	return intent
}

func hasPipelineInputArtifact(inputs []pipeline.InputArtifact, name, producer string) bool {
	for _, input := range inputs {
		if input.Artifact.Name == name && input.ProducerJob == producer {
			return true
		}
	}
	return false
}
