package generate

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
)

// TestEdgeCase_EmptyTargetModules tests generation with empty slice of target modules
func TestEdgeCase_EmptyTargetModules(t *testing.T) {
	pipeline := newFixtureScenario(t, "basic").withTargets([]*discovery.Module{}...).generate()

	if len(pipeline.Jobs) == 0 {
		t.Log("Empty target modules generated 0 jobs - this may be correct if empty means 'no changes'")
	}
}

// TestEdgeCase_NilTargetModules tests generation with nil target modules
func TestEdgeCase_NilTargetModules(t *testing.T) {
	scenario := newFixtureScenario(t, "basic")
	pipeline := scenario.generate()
	assertPipeline(t, pipeline).jobCount(len(scenario.fixture.Modules) * 2)
}

// TestEdgeCase_SingleModuleNoDependencies tests single module with no dependencies
func TestEdgeCase_SingleModuleNoDependencies(t *testing.T) {
	scenario := newFixtureScenario(t, "basic").withTargetNames("vpc")
	var targetModules []*discovery.Module
	for _, module := range scenario.targets {
		if module.Get("environment") == "stage" {
			targetModules = append(targetModules, module)
		}
	}
	if len(targetModules) == 0 {
		for _, module := range scenario.fixture.Modules {
			if module.Get("module") == "vpc" && module.Get("environment") == "stage" {
				targetModules = append(targetModules, module)
			}
		}
	}
	pipeline := scenario.withTargets(targetModules...).generate()

	assertPipeline(t, pipeline).
		jobCount(2).
		stageCount(2).
		job("plan-platform-stage-eu-central-1-vpc").
		noNeeds()
}

// TestEdgeCase_SingleModuleWithDependencies tests single module that has dependencies
func TestEdgeCase_SingleModuleWithDependencies(t *testing.T) {
	pipeline := newFixtureScenario(t, "basic").
		withTargetNames("app").
		generate()

	assertPipeline(t, pipeline).
		jobCount(2).
		job("plan-platform-stage-eu-central-1-app").
		noNeeds()
}

// TestEdgeCase_AllModulesIndependent tests modules with no dependencies between them
func TestEdgeCase_AllModulesIndependent(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "stage", "eu-central-1", "a"),
		discovery.TestModule("svc", "stage", "eu-central-1", "b"),
		discovery.TestModule("svc", "stage", "eu-central-1", "c"),
	}

	pipeline := newGeneratorScenario(t).
		withExecution(func(cfg *execution.Config) { cfg.PlanEnabled = true }).
		withModules(modules...).
		withDependencies(map[string][]string{
			"svc/stage/eu-central-1/a": {},
			"svc/stage/eu-central-1/b": {},
			"svc/stage/eu-central-1/c": {},
		}).
		generate()

	assertPipeline(t, pipeline).stageCount(2)
	assertPipeline(t, pipeline).job("plan-svc-stage-eu-central-1-a").noNeeds()
	assertPipeline(t, pipeline).job("plan-svc-stage-eu-central-1-b").noNeeds()
	assertPipeline(t, pipeline).job("plan-svc-stage-eu-central-1-c").noNeeds()
}

// TestEdgeCase_DeepDependencyChain tests a long chain of dependencies
func TestEdgeCase_DeepDependencyChain(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "stage", "eu-central-1", "a"),
		discovery.TestModule("svc", "stage", "eu-central-1", "b"),
		discovery.TestModule("svc", "stage", "eu-central-1", "c"),
		discovery.TestModule("svc", "stage", "eu-central-1", "d"),
		discovery.TestModule("svc", "stage", "eu-central-1", "e"),
	}

	pipeline := newGeneratorScenario(t).
		withExecution(func(cfg *execution.Config) { cfg.PlanEnabled = true }).
		withModules(modules...).
		withDependencies(map[string][]string{
			"svc/stage/eu-central-1/a": {},
			"svc/stage/eu-central-1/b": {"svc/stage/eu-central-1/a"},
			"svc/stage/eu-central-1/c": {"svc/stage/eu-central-1/b"},
			"svc/stage/eu-central-1/d": {"svc/stage/eu-central-1/c"},
			"svc/stage/eu-central-1/e": {"svc/stage/eu-central-1/d"},
		}).
		generate()

	assertPipeline(t, pipeline).
		stageCount(10).
		jobStageBefore("apply-svc-stage-eu-central-1-a", "apply-svc-stage-eu-central-1-b").
		jobStageBefore("apply-svc-stage-eu-central-1-b", "apply-svc-stage-eu-central-1-c").
		jobStageBefore("apply-svc-stage-eu-central-1-c", "apply-svc-stage-eu-central-1-d").
		jobStageBefore("apply-svc-stage-eu-central-1-d", "apply-svc-stage-eu-central-1-e")
}

// TestEdgeCase_DiamondDependency tests diamond dependency pattern
func TestEdgeCase_DiamondDependency(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "stage", "eu-central-1", "a"),
		discovery.TestModule("svc", "stage", "eu-central-1", "b"),
		discovery.TestModule("svc", "stage", "eu-central-1", "c"),
		discovery.TestModule("svc", "stage", "eu-central-1", "d"),
	}

	pipeline := newGeneratorScenario(t).
		withExecution(func(cfg *execution.Config) { cfg.PlanEnabled = true }).
		withModules(modules...).
		withDependencies(map[string][]string{
			"svc/stage/eu-central-1/a": {},
			"svc/stage/eu-central-1/b": {"svc/stage/eu-central-1/a"},
			"svc/stage/eu-central-1/c": {"svc/stage/eu-central-1/a"},
			"svc/stage/eu-central-1/d": {"svc/stage/eu-central-1/b", "svc/stage/eu-central-1/c"},
		}).
		generate()

	assertPipeline(t, pipeline).
		job("plan-svc-stage-eu-central-1-d").
		hasNeed("apply-svc-stage-eu-central-1-b").
		hasNeed("apply-svc-stage-eu-central-1-c")
	jobB := mustJob(t, pipeline, "apply-svc-stage-eu-central-1-b")
	jobC := mustJob(t, pipeline, "apply-svc-stage-eu-central-1-c")
	if jobB.Stage != jobC.Stage {
		t.Errorf("B and C should be in same stage, got B=%s C=%s", jobB.Stage, jobC.Stage)
	}
}

// TestEdgeCase_PartialChainChanged tests when middle of chain changes
func TestEdgeCase_PartialChainChanged(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "stage", "eu-central-1", "a"),
		discovery.TestModule("svc", "stage", "eu-central-1", "b"),
		discovery.TestModule("svc", "stage", "eu-central-1", "c"),
	}

	pipeline := newGeneratorScenario(t).
		withExecution(func(cfg *execution.Config) { cfg.PlanEnabled = true }).
		withModules(modules...).
		withDependencies(map[string][]string{
			"svc/stage/eu-central-1/a": {},
			"svc/stage/eu-central-1/b": {"svc/stage/eu-central-1/a"},
			"svc/stage/eu-central-1/c": {"svc/stage/eu-central-1/b"},
		}).
		withTargets(modules[1]).
		generate()

	assertPipeline(t, pipeline).
		jobCount(2).
		hasJob("plan-svc-stage-eu-central-1-b").
		hasJob("apply-svc-stage-eu-central-1-b").
		noJob("plan-svc-stage-eu-central-1-a").
		noJob("plan-svc-stage-eu-central-1-c")
	assertPipeline(t, pipeline).
		job("plan-svc-stage-eu-central-1-b").
		noNeeds()
}

// TestEdgeCase_PlanOnlyWithNoPlanEnabled tests conflict: plan_only=true but plan_enabled=false
func TestEdgeCase_PlanOnlyWithNoPlanEnabled(t *testing.T) {
	pipeline := newFixtureScenario(t, "basic").
		withConfig(func(cfg *Config) {
			cfg.PlanOnly = true
		}).
		withExecution(func(cfg *execution.Config) { cfg.PlanEnabled = false }).
		generate()

	if len(pipeline.Jobs) != 0 {
		t.Logf("Got %d jobs with PlanOnly=true, PlanEnabled=false", len(pipeline.Jobs))
	}
}

// TestEdgeCase_AutoApproveMode tests auto_approve flag
func TestEdgeCase_AutoApproveMode(t *testing.T) {
	pipeline := newFixtureScenario(t, "basic").
		withConfig(func(cfg *Config) { cfg.AutoApprove = true }).
		generate()

	for jobName, job := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "apply-") {
			if job.When == "manual" {
				t.Errorf("Apply job %s should not be manual when auto_approve=true", jobName)
			}
		}
	}
}

// TestEdgeCase_ManualApproveMode tests manual approval (default)
func TestEdgeCase_ManualApproveMode(t *testing.T) {
	pipeline := newFixtureScenario(t, "basic").
		withConfig(func(cfg *Config) { cfg.AutoApprove = false }).
		generate()

	for jobName, job := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "apply-") {
			if job.When != "manual" {
				t.Errorf("Apply job %s should be manual when auto_approve=false, got %q", jobName, job.When)
			}
		}
	}
}

// TestEdgeCase_ChangedOnlyNoChanges simulates --changed-only with no actual changes
func TestEdgeCase_ChangedOnlyNoChanges(t *testing.T) {
	pipeline := newFixtureScenario(t, "basic").withTargets([]*discovery.Module{}...).generate()

	if len(pipeline.Jobs) == 0 {
		t.Log("Empty changes resulted in 0 jobs - may need to handle this case in CLI")
	} else {
		t.Logf("Empty changes resulted in %d jobs (fallback to all)", len(pipeline.Jobs))
	}
}

// TestEdgeCase_ModuleWithSelfReference tests (edge case) module referencing itself
func TestEdgeCase_ModuleWithSelfReference(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "stage", "eu-central-1", "a"),
	}

	// Self-reference (should be ignored or handled gracefully)
	deps := map[string]*parser.ModuleDependencies{
		"svc/stage/eu-central-1/a": {DependsOn: []string{"svc/stage/eu-central-1/a"}},
	}

	depGraph := graph.BuildFromDependencies(modules, deps)

	glCfg := &Config{}
	execCfg := execution.Config{
		Binary:      "terraform",
		InitEnabled: true,
		PlanEnabled: true,
		PlanMode:    execution.PlanModeStandard,
		Parallelism: 4,
	}

	generator := NewGenerator(glCfg, execCfg, nil, depGraph, modules)
	result, err := generator.Generate(modules)
	if err != nil {
		// Self-reference might cause cycle detection
		t.Logf("Self-reference caused error (expected): %v", err)
		return
	}

	pipeline, ok := result.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}
	assertPipeline(t, pipeline).jobCount(2)
}

// TestEdgeCase_SpecialCharactersInModuleName tests module names with special chars
func TestEdgeCase_SpecialCharactersInModuleName(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("my-svc", "stage-01", "eu-central-1", "vpc-main"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"my-svc/stage-01/eu-central-1/vpc-main": {DependsOn: []string{}},
	}

	depGraph := graph.BuildFromDependencies(modules, deps)

	glCfg := &Config{}
	execCfg := execution.Config{
		Binary:      "terraform",
		InitEnabled: true,
		PlanEnabled: true,
		PlanMode:    execution.PlanModeStandard,
		Parallelism: 4,
	}

	generator := NewGenerator(glCfg, execCfg, nil, depGraph, modules)
	result, err := generator.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}
	assertPipeline(t, pipeline).hasJob("plan-my-svc-stage-01-eu-central-1-vpc-main")
}

// TestEdgeCase_VeryLongModulePath tests handling of long module paths
func TestEdgeCase_VeryLongModulePath(t *testing.T) {
	longService := "very-long-service-name-for-testing"
	longEnv := "development-environment"
	longRegion := "eu-central-1"
	longModule := "application-database-migration"

	modules := []*discovery.Module{
		discovery.TestModule(longService, longEnv, longRegion, longModule),
	}

	deps := map[string]*parser.ModuleDependencies{
		longService + "/" + longEnv + "/" + longRegion + "/" + longModule: {DependsOn: []string{}},
	}

	depGraph := graph.BuildFromDependencies(modules, deps)

	glCfg := &Config{}
	execCfg := execution.Config{
		Binary:      "terraform",
		InitEnabled: true,
		PlanEnabled: true,
		PlanMode:    execution.PlanModeStandard,
		Parallelism: 4,
	}

	generator := NewGenerator(glCfg, execCfg, nil, depGraph, modules)
	result, err := generator.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}
	assertPipeline(t, pipeline).jobCount(2)
}
