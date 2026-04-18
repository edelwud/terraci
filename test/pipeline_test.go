package test

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/pipeline"
)

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

	if len(ir.Levels) == 0 {
		t.Fatal("expected at least 1 level")
	}

	totalModules := 0
	for _, level := range ir.Levels {
		totalModules += len(level.Modules)
	}
	if totalModules != 2 {
		t.Errorf("expected 2 modules, got %d", totalModules)
	}

	for _, level := range ir.Levels {
		for _, mj := range level.Modules {
			if mj.Plan == nil {
				t.Errorf("module %s missing plan job", mj.Module.ID())
			}
			if mj.Apply == nil {
				t.Errorf("module %s missing apply job", mj.Module.ID())
			}
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
		PlanEnabled: true,
		PlanOnly:    true,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	for _, level := range ir.Levels {
		for _, mj := range level.Modules {
			if mj.Plan == nil {
				t.Error("plan job should exist in plan-only mode")
			}
			if mj.Apply != nil {
				t.Error("apply job should not exist in plan-only mode")
			}
		}
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

	contributions := []*pipeline.Contribution{{
		Jobs: []pipeline.ContributedJob{{
			Name:          "policy-check",
			Phase:         pipeline.PhasePostPlan,
			Commands:      []string{"terraci policy check"},
			DependsOnPlan: true,
		}},
	}}

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

	if len(ir.Jobs) != 1 {
		t.Fatalf("expected 1 contributed job, got %d", len(ir.Jobs))
	}
	if ir.Jobs[0].Name != "policy-check" {
		t.Errorf("expected policy-check, got %s", ir.Jobs[0].Name)
	}

	// policy-check should depend on the plan job
	hasPlanDep := false
	for _, dep := range ir.Jobs[0].Dependencies {
		if dep == ir.Levels[0].Modules[0].Plan.Name {
			hasPlanDep = true
		}
	}
	if !hasPlanDep {
		t.Error("policy-check should depend on the plan job")
	}
}

func TestPipelineBuild_PhaseFinalize(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "prod", "eu", "app"),
	}
	deps := map[string]*parser.ModuleDependencies{modules[0].ID(): {}}
	depGraph := graph.BuildFromDependencies(modules, deps)
	index := discovery.NewModuleIndex(modules)

	contributions := []*pipeline.Contribution{
		{Jobs: []pipeline.ContributedJob{{
			Name: "policy-check", Phase: pipeline.PhasePostPlan,
			Commands: []string{"check"}, DependsOnPlan: true,
		}}},
		{Jobs: []pipeline.ContributedJob{{
			Name: "terraci-summary", Phase: pipeline.PhaseFinalize,
			Commands: []string{"summary"}, DependsOnPlan: true,
		}}},
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

	// Find finalize job
	var finalizeJob *pipeline.Job
	for i := range ir.Jobs {
		if ir.Jobs[i].Phase == pipeline.PhaseFinalize {
			finalizeJob = &ir.Jobs[i]
			break
		}
	}
	if finalizeJob == nil {
		t.Fatal("missing finalize job")
	}

	// Should depend on plan job(s) AND policy-check
	hasPolicyDep := false
	hasPlanDep := false
	for _, dep := range finalizeJob.Dependencies {
		if dep == "policy-check" {
			hasPolicyDep = true
		}
		if dep == ir.Levels[0].Modules[0].Plan.Name {
			hasPlanDep = true
		}
	}
	if !hasPolicyDep {
		t.Error("finalize job should depend on policy-check")
	}
	if !hasPlanDep {
		t.Error("finalize job should depend on plan jobs")
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

	// With a chain of 3, we expect multiple levels
	if len(ir.Levels) < 2 {
		t.Errorf("expected at least 2 levels for a dependency chain, got %d", len(ir.Levels))
	}
}
