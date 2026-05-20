package test

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/pipeline"
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
	index := discovery.NewModuleIndex(modules)

	ir, err := pipeline.Build(pipeline.BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script: pipeline.ScriptConfig{
			InitEnabled: true,
			PlanEnabled: true,
		},
		PlanEnabled: true,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if got := ir.ModuleCount(); got != 2 {
		t.Errorf("expected 2 modules, got %d", got)
	}

	for _, mod := range modules {
		if findPipelineJob(ir, pipeline.JobName(pipeline.JobKindPlan, mod)) == nil {
			t.Errorf("module %s missing plan job", mod.ID())
		}
		if findPipelineJob(ir, pipeline.JobName(pipeline.JobKindApply, mod)) == nil {
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
	index := discovery.NewModuleIndex(modules)

	ir, err := pipeline.Build(pipeline.BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script: pipeline.ScriptConfig{
			PlanEnabled: true,
		},
		PlanEnabled:  true,
		Requirements: pipeline.BuildRequirements{PlanOnly: true},
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if findPipelineJob(ir, pipeline.JobName(pipeline.JobKindPlan, modules[0])) == nil {
		t.Error("plan job should exist in plan-only mode")
	}
	if findPipelineJob(ir, pipeline.JobName(pipeline.JobKindApply, modules[0])) != nil {
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
	index := discovery.NewModuleIndex(modules)

	contributions := []*pipeline.Contribution{mustPipelineContribution(t, pipeline.ContributedJobOptions{
		Name:     "policy-check",
		Commands: []string{"terraci policy check --format text"},
		Consumes: []pipeline.ResourceRequest{
			pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
		},
	})}

	ir, err := pipeline.Build(pipeline.BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script: pipeline.ScriptConfig{
			PlanEnabled: true,
		},
		Contributions: contributions,
		PlanEnabled:   true,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	policyJob := findPipelineJob(ir, "policy-check")
	if policyJob == nil {
		t.Fatal("expected policy-check job")
	}

	// policy-check should depend on the plan job
	planName := pipeline.JobName(pipeline.JobKindPlan, modules[0])
	if !hasPipelineDependency(policyJob.Dependencies, planName) {
		t.Error("policy-check should depend on the plan job")
	}
	if !hasPipelineInputArtifact(policyJob.InputArtifacts, pipeline.PlanArtifactName(planName), planName) {
		t.Error("policy-check should restore the plan artifact")
	}
}

func TestPipelineBuild_SummaryDependsThroughResources(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "prod", "eu", "app"),
	}
	deps := map[string]*parser.ModuleDependencies{modules[0].ID(): {}}
	depGraph := graph.BuildFromDependencies(modules, deps)
	index := discovery.NewModuleIndex(modules)

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

	ir, err := pipeline.Build(pipeline.BuildOptions{
		DepGraph: depGraph, TargetModules: modules, AllModules: modules,
		ModuleIndex:   index,
		Script:        pipeline.ScriptConfig{PlanEnabled: true},
		Contributions: contributions, PlanEnabled: true,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	var summaryJob *pipeline.Job
	for i := range ir.Jobs {
		if ir.Jobs[i].Name == "terraci-summary" {
			summaryJob = &ir.Jobs[i]
			break
		}
	}
	if summaryJob == nil {
		t.Fatal("missing summary job")
	}

	planName := pipeline.JobName(pipeline.JobKindPlan, modules[0])
	if !hasPipelineDependency(summaryJob.Dependencies, "policy-check") {
		t.Error("summary job should depend on policy-check report")
	}
	if !hasPipelineDependency(summaryJob.Dependencies, planName) {
		t.Error("summary job should depend on plan jobs")
	}
	if !hasPipelineInputArtifact(summaryJob.InputArtifacts, pipeline.ResultArtifactName("policy-check"), "policy-check") {
		t.Error("summary job should restore policy report artifact")
	}
	if !hasPipelineInputArtifact(summaryJob.InputArtifacts, pipeline.PlanArtifactName(planName), planName) {
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
	index := discovery.NewModuleIndex(modules)

	ir, err := pipeline.Build(pipeline.BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script: pipeline.ScriptConfig{
			PlanEnabled: true,
		},
		PlanEnabled: true,
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

	eksPlan := findPipelineJob(ir, pipeline.JobName(pipeline.JobKindPlan, modules[1]))
	if eksPlan == nil {
		t.Fatal("missing eks plan job")
	}
	if !hasPipelineDependency(eksPlan.Dependencies, pipeline.JobName(pipeline.JobKindApply, modules[0])) {
		t.Error("dependent module plan should depend on upstream apply in full run mode")
	}
}

func findPipelineJob(ir *pipeline.IR, name string) *pipeline.Job {
	for i := range ir.Jobs {
		if ir.Jobs[i].Name == name {
			return &ir.Jobs[i]
		}
	}
	return nil
}

func hasPipelineDependency(deps []pipeline.JobDependency, name string) bool {
	for _, dep := range deps {
		if dep.Job == name {
			return true
		}
	}
	return false
}

func hasPipelineInputArtifact(inputs []pipeline.InputArtifact, name, producer string) bool {
	for _, input := range inputs {
		if input.Artifact.Name == name && input.ProducerJob == producer {
			return true
		}
	}
	return false
}
