package generate

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
)

// TestFixture_Basic tests basic pipeline generation using real terraform fixtures
func TestFixture_Basic(t *testing.T) {
	scenario := newFixtureScenario(t, "basic")

	// Should discover 6 modules (5 stage + 1 prod)
	if len(scenario.fixture.Modules) != 6 {
		t.Errorf("expected 6 modules, got %d", len(scenario.fixture.Modules))
	}

	pipeline := scenario.generate()

	assertPipeline(t, pipeline).
		jobCount(12).
		hasJob("plan-platform-stage-eu-central-1-vpc").
		hasJob("apply-platform-stage-eu-central-1-vpc").
		hasJob("plan-platform-stage-eu-central-1-eks").
		hasJob("apply-platform-stage-eu-central-1-eks").
		hasJob("plan-platform-prod-eu-central-1-vpc").
		hasJob("apply-platform-prod-eu-central-1-vpc")
}

// TestFixture_BasicDependencies tests that dependencies are correctly resolved
func TestFixture_BasicDependencies(t *testing.T) {
	pipeline := newFixtureScenario(t, "basic").generate()

	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-eks").
		hasNeed("apply-platform-stage-eu-central-1-vpc")
	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-rds").
		hasNeed("apply-platform-stage-eu-central-1-vpc")
	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-app").
		hasNeed("apply-platform-stage-eu-central-1-eks").
		hasNeed("apply-platform-stage-eu-central-1-rds").
		hasNeed("apply-platform-stage-eu-central-1-s3")
	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-vpc").
		noNeeds()
}

// TestFixture_PlanOnly tests plan-only mode with fixtures
func TestFixture_PlanOnly(t *testing.T) {
	pipeline := newFixtureScenario(t, "basic").
		withConfig(func(cfg *Config) {
			cfg.PlanOnly = true
			cfg.PlanEnabled = true
		}).
		generate()

	assertPipeline(t, pipeline).jobCount(6)

	// No apply jobs should exist
	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "apply-") {
			t.Errorf("unexpected apply job in plan-only mode: %s", jobName)
		}
	}

	// No apply stages should exist
	for _, stage := range pipeline.Stages {
		if strings.Contains(stage, "-apply-") {
			t.Errorf("unexpected apply stage in plan-only mode: %s", stage)
		}
	}

	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-eks").
		hasNeed("plan-platform-stage-eu-central-1-vpc").
		noNeedWithPrefix("apply-")
}

// TestFixture_ChangedOnly tests changed-only mode with fixtures
func TestFixture_ChangedOnly(t *testing.T) {
	scenario := newFixtureScenario(t, "basic").
		withTargetNames("eks", "app")
	var stageChangedModules []*discovery.Module
	for _, module := range scenario.targets {
		if module.Get("environment") == "stage" {
			stageChangedModules = append(stageChangedModules, module)
		}
	}
	pipeline := scenario.withTargets(stageChangedModules...).generate()

	assertPipeline(t, pipeline).
		jobCount(4).
		hasJob("plan-platform-stage-eu-central-1-eks").
		hasJob("apply-platform-stage-eu-central-1-eks").
		hasJob("plan-platform-stage-eu-central-1-app").
		hasJob("apply-platform-stage-eu-central-1-app").
		noJob("plan-platform-stage-eu-central-1-vpc").
		noJob("apply-platform-stage-eu-central-1-vpc").
		noJob("plan-platform-stage-eu-central-1-s3").
		noJob("apply-platform-stage-eu-central-1-s3")
	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-eks").
		noNeed("apply-platform-stage-eu-central-1-vpc")
	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-app").
		hasNeed("apply-platform-stage-eu-central-1-eks")
	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-app").
		noNeed("apply-platform-stage-eu-central-1-s3").
		noNeed("apply-platform-stage-eu-central-1-rds")
}

// TestFixture_ChangedOnlyPlanOnly tests combined changed-only and plan-only modes
func TestFixture_ChangedOnlyPlanOnly(t *testing.T) {
	scenario := newFixtureScenario(t, "basic").
		withConfig(func(cfg *Config) {
			cfg.PlanOnly = true
			cfg.PlanEnabled = true
		}).
		withTargetNames("eks")
	var changedModules []*discovery.Module
	for _, module := range scenario.targets {
		if module.Get("environment") == "stage" {
			changedModules = append(changedModules, module)
		}
	}
	pipeline := scenario.withTargets(changedModules...).generate()

	assertPipeline(t, pipeline).
		jobCount(1).
		hasJob("plan-platform-stage-eu-central-1-eks")

	// No apply jobs
	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "apply-") {
			t.Errorf("unexpected apply job: %s", jobName)
		}
	}

	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-eks").
		noNeeds()
}

// TestFixture_EnvironmentFilter tests filtering by environment
func TestFixture_EnvironmentFilter(t *testing.T) {
	scenario := newFixtureScenario(t, "basic").withEnvironment("stage")

	if len(scenario.targets) != 5 {
		t.Errorf("expected 5 stage modules, got %d", len(scenario.targets))
	}

	pipeline := scenario.generate()

	assertPipeline(t, pipeline).
		jobCount(10).
		noJob("plan-platform-prod-eu-central-1-vpc").
		noJob("apply-platform-prod-eu-central-1-vpc")
}

// TestFixture_Submodules tests submodule support with fixtures
func TestFixture_Submodules(t *testing.T) {
	scenario := newFixtureScenario(t, "submodules")

	// Should discover 3 modules (base + ec2/web + ec2/worker)
	if len(scenario.fixture.Modules) != 3 {
		t.Errorf("expected 3 modules, got %d", len(scenario.fixture.Modules))
		for _, m := range scenario.fixture.Modules {
			t.Logf("  module: %s", m.ID())
		}
	}

	pipeline := scenario.generate()

	assertPipeline(t, pipeline).
		jobCount(6).
		hasJob("plan-svc-stage-eu-central-1-ec2-web").
		hasJob("apply-svc-stage-eu-central-1-ec2-web").
		hasJob("plan-svc-stage-eu-central-1-ec2-worker").
		hasJob("apply-svc-stage-eu-central-1-ec2-worker")
	assertPipeline(t, pipeline).
		job("plan-svc-stage-eu-central-1-ec2-web").
		hasNeed("apply-svc-stage-eu-central-1-base")
	assertPipeline(t, pipeline).
		job("plan-svc-stage-eu-central-1-ec2-worker").
		hasNeed("apply-svc-stage-eu-central-1-base")
}

// TestFixture_CyclicDependencies tests cycle detection with fixtures
func TestFixture_CyclicDependencies(t *testing.T) {
	fixture := LoadFixture(t, "cyclic")

	// Should detect cycles
	cycles := fixture.DepGraph.DetectCycles()
	if len(cycles) == 0 {
		t.Error("expected to detect cycles, found none")
	}

	// Topological sort should fail
	_, err := fixture.DepGraph.TopologicalSort()
	if err == nil {
		t.Error("expected topological sort to fail with cycles")
	}
}

// TestFixture_ApplyDependsOnPlan tests that apply jobs depend on their plan jobs
func TestFixture_ApplyDependsOnPlan(t *testing.T) {
	scenario := newFixtureScenario(t, "basic")
	pipeline := scenario.generate()

	// Each apply job should depend on its own plan job
	for _, module := range scenario.fixture.Modules {
		moduleID := strings.ReplaceAll(module.ID(), "/", "-")
		applyJobName := "apply-" + moduleID
		planJobName := "plan-" + moduleID

		assertPipeline(t, pipeline).
			job(applyJobName).
			hasNeed(planJobName)
	}
}

// TestFixture_NoPlanEnabled tests pipeline without plan stage
func TestFixture_NoPlanEnabled(t *testing.T) {
	pipeline := newFixtureScenario(t, "basic").
		withConfig(func(cfg *Config) { cfg.PlanEnabled = false }).
		generate()

	assertPipeline(t, pipeline).jobCount(6)

	// No plan jobs
	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "plan-") {
			t.Errorf("expected no plan jobs, got %s", jobName)
		}
	}

	// No plan stages
	for _, stage := range pipeline.Stages {
		if strings.Contains(stage, "-plan-") {
			t.Errorf("unexpected plan stage: %s", stage)
		}
	}
}

// TestFixture_StageOrder tests that stages are in correct dependency order
func TestFixture_StageOrder(t *testing.T) {
	pipeline := newFixtureScenario(t, "basic").
		withEnvironment("stage").
		generate()
	assertPipeline(t, pipeline).
		jobStageBefore("apply-platform-stage-eu-central-1-vpc", "apply-platform-stage-eu-central-1-eks").
		jobStageBefore("apply-platform-stage-eu-central-1-s3", "apply-platform-stage-eu-central-1-app").
		jobStageBefore("apply-platform-stage-eu-central-1-eks", "apply-platform-stage-eu-central-1-app").
		jobStageBefore("apply-platform-stage-eu-central-1-rds", "apply-platform-stage-eu-central-1-app")
}
