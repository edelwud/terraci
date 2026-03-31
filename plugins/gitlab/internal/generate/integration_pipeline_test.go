package generate

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
)

// createTestModules creates a standard test module set with dependencies:
// Level 0: vpc, s3
// Level 1: eks (depends on vpc), rds (depends on vpc)
// Level 2: app (depends on eks, rds, s3)
func createTestModules() []*discovery.Module {
	return []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "s3"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
		discovery.TestModule("platform", "stage", "eu-central-1", "rds"),
		discovery.TestModule("platform", "stage", "eu-central-1", "app"),
	}
}

func TestPipelineGeneration_Basic(t *testing.T) {
	modules := createTestModules()
	pipeline := newGeneratorScenario(t).
		withConfig(func(cfg *Config) {
			cfg.PlanEnabled = true
			cfg.AutoApprove = false
		}).
		withModules(modules...).
		withDependencies(map[string][]string{
			"platform/stage/eu-central-1/vpc": {},
			"platform/stage/eu-central-1/s3":  {},
			"platform/stage/eu-central-1/eks": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/rds": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/app": {
				"platform/stage/eu-central-1/eks",
				"platform/stage/eu-central-1/rds",
				"platform/stage/eu-central-1/s3",
			},
		}).
		generate()

	hasplanStage := false
	hasApplyStage := false
	for _, stage := range pipeline.Stages {
		if strings.HasPrefix(stage, "deploy-plan-") {
			hasplanStage = true
		}
		if strings.HasPrefix(stage, "deploy-apply-") {
			hasApplyStage = true
		}
	}

	if !hasplanStage {
		t.Error("Expected plan stages in pipeline")
	}
	if !hasApplyStage {
		t.Error("Expected apply stages in pipeline")
	}

	for _, jobName := range []string{
		"plan-platform-stage-eu-central-1-vpc",
		"plan-platform-stage-eu-central-1-s3",
		"plan-platform-stage-eu-central-1-eks",
		"plan-platform-stage-eu-central-1-rds",
		"plan-platform-stage-eu-central-1-app",
		"apply-platform-stage-eu-central-1-vpc",
		"apply-platform-stage-eu-central-1-s3",
		"apply-platform-stage-eu-central-1-eks",
		"apply-platform-stage-eu-central-1-rds",
		"apply-platform-stage-eu-central-1-app",
	} {
		assertPipeline(t, pipeline).hasJob(jobName)
	}
}

func TestPipelineGeneration_PlanOnly(t *testing.T) {
	modules := createTestModules()
	pipeline := newGeneratorScenario(t).
		withConfig(func(cfg *Config) {
			cfg.PlanEnabled = true
			cfg.PlanOnly = true
		}).
		withModules(modules...).
		withDependencies(map[string][]string{
			"platform/stage/eu-central-1/vpc": {},
			"platform/stage/eu-central-1/s3":  {},
			"platform/stage/eu-central-1/eks": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/rds": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/app": {
				"platform/stage/eu-central-1/eks",
				"platform/stage/eu-central-1/rds",
				"platform/stage/eu-central-1/s3",
			},
		}).
		generate()

	assertPipeline(t, pipeline).noStageWithFragment("-apply-")

	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "apply-") {
			t.Errorf("Unexpected apply job in plan-only mode: %s", jobName)
		}
	}

	for _, jobName := range []string{
		"plan-platform-stage-eu-central-1-vpc",
		"plan-platform-stage-eu-central-1-s3",
		"plan-platform-stage-eu-central-1-eks",
		"plan-platform-stage-eu-central-1-rds",
		"plan-platform-stage-eu-central-1-app",
	} {
		assertPipeline(t, pipeline).hasJob(jobName)
	}
}

func TestPipelineGeneration_PlanOnlyNeeds(t *testing.T) {
	modules := createTestModules()
	pipeline := newGeneratorScenario(t).
		withConfig(func(cfg *Config) {
			cfg.PlanEnabled = true
			cfg.PlanOnly = true
		}).
		withModules(modules...).
		withDependencies(map[string][]string{
			"platform/stage/eu-central-1/vpc": {},
			"platform/stage/eu-central-1/s3":  {},
			"platform/stage/eu-central-1/eks": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/rds": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/app": {
				"platform/stage/eu-central-1/eks",
				"platform/stage/eu-central-1/rds",
				"platform/stage/eu-central-1/s3",
			},
		}).
		generate()

	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-eks").
		hasNeed("plan-platform-stage-eu-central-1-vpc").
		noNeedWithPrefix("apply-")
	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-app").
		hasNeed("plan-platform-stage-eu-central-1-eks").
		hasNeed("plan-platform-stage-eu-central-1-rds").
		hasNeed("plan-platform-stage-eu-central-1-s3").
		noNeedWithPrefix("apply-")
}

func TestPipelineGeneration_ChangedOnlyFilteredNeeds(t *testing.T) {
	modules := createTestModules()
	pipeline := newGeneratorScenario(t).
		withConfig(func(cfg *Config) { cfg.PlanEnabled = true }).
		withModules(modules...).
		withDependencies(map[string][]string{
			"platform/stage/eu-central-1/vpc": {},
			"platform/stage/eu-central-1/s3":  {},
			"platform/stage/eu-central-1/eks": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/rds": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/app": {
				"platform/stage/eu-central-1/eks",
				"platform/stage/eu-central-1/rds",
				"platform/stage/eu-central-1/s3",
			},
		}).
		withTargets(modules[2], modules[4]).
		generate()

	assertPipeline(t, pipeline).jobCount(4)

	for _, jobName := range []string{
		"plan-platform-stage-eu-central-1-vpc",
		"apply-platform-stage-eu-central-1-vpc",
		"plan-platform-stage-eu-central-1-s3",
		"apply-platform-stage-eu-central-1-s3",
		"plan-platform-stage-eu-central-1-rds",
		"apply-platform-stage-eu-central-1-rds",
	} {
		assertPipeline(t, pipeline).noJob(jobName)
	}

	for _, jobName := range []string{
		"plan-platform-stage-eu-central-1-eks",
		"apply-platform-stage-eu-central-1-eks",
		"plan-platform-stage-eu-central-1-app",
		"apply-platform-stage-eu-central-1-app",
	} {
		assertPipeline(t, pipeline).hasJob(jobName)
	}

	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-app").
		hasNeed("apply-platform-stage-eu-central-1-eks").
		noNeed("apply-platform-stage-eu-central-1-s3").
		noNeed("apply-platform-stage-eu-central-1-rds")
	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-eks").
		noNeed("apply-platform-stage-eu-central-1-vpc")
}

func TestPipelineGeneration_ChangedOnlyPlanOnly(t *testing.T) {
	modules := createTestModules()
	pipeline := newGeneratorScenario(t).
		withConfig(func(cfg *Config) {
			cfg.PlanEnabled = true
			cfg.PlanOnly = true
		}).
		withModules(modules...).
		withDependencies(map[string][]string{
			"platform/stage/eu-central-1/vpc": {},
			"platform/stage/eu-central-1/s3":  {},
			"platform/stage/eu-central-1/eks": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/rds": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/app": {
				"platform/stage/eu-central-1/eks",
				"platform/stage/eu-central-1/rds",
				"platform/stage/eu-central-1/s3",
			},
		}).
		withTargets(modules[2], modules[4]).
		generate()

	assertPipeline(t, pipeline).jobCount(2)

	// No apply jobs
	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "apply-") {
			t.Errorf("Unexpected apply job in plan-only mode: %s", jobName)
		}
	}

	assertPipeline(t, pipeline).
		job("plan-platform-stage-eu-central-1-app").
		hasNeed("plan-platform-stage-eu-central-1-eks").
		noNeedWithPrefix("apply-")
}

func TestPipelineGeneration_ApplyDependsOnPlan(t *testing.T) {
	modules := createTestModules()
	pipeline := newGeneratorScenario(t).
		withConfig(func(cfg *Config) { cfg.PlanEnabled = true }).
		withModules(modules...).
		withDependencies(map[string][]string{
			"platform/stage/eu-central-1/vpc": {},
			"platform/stage/eu-central-1/s3":  {},
			"platform/stage/eu-central-1/eks": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/rds": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/app": {
				"platform/stage/eu-central-1/eks",
				"platform/stage/eu-central-1/rds",
				"platform/stage/eu-central-1/s3",
			},
		}).
		generate()

	for _, module := range modules {
		moduleID := strings.ReplaceAll(module.ID(), "/", "-")
		assertPipeline(t, pipeline).
			job("apply-" + moduleID).
			hasNeed("plan-" + moduleID)
	}
}

func TestPipelineGeneration_NoPlanEnabled(t *testing.T) {
	modules := createTestModules()
	pipeline := newGeneratorScenario(t).
		withConfig(func(cfg *Config) { cfg.PlanEnabled = false }).
		withModules(modules...).
		withDependencies(map[string][]string{
			"platform/stage/eu-central-1/vpc": {},
			"platform/stage/eu-central-1/s3":  {},
			"platform/stage/eu-central-1/eks": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/rds": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/app": {
				"platform/stage/eu-central-1/eks",
				"platform/stage/eu-central-1/rds",
				"platform/stage/eu-central-1/s3",
			},
		}).
		generate()

	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "plan-") {
			t.Errorf("Unexpected plan job when PlanEnabled=false: %s", jobName)
		}
	}

	assertPipeline(t, pipeline).noStageWithFragment("-plan-")

	for jobName, job := range pipeline.Jobs {
		for _, need := range job.Needs {
			if strings.HasPrefix(need.Job, "plan-") {
				t.Errorf("Apply job %s should not depend on plan job when PlanEnabled=false: %s", jobName, need.Job)
			}
		}
	}
}

func TestPipelineGeneration_DependencyOrder(t *testing.T) {
	modules := createTestModules()
	pipeline := newGeneratorScenario(t).
		withConfig(func(cfg *Config) { cfg.PlanEnabled = true }).
		withModules(modules...).
		withDependencies(map[string][]string{
			"platform/stage/eu-central-1/vpc": {},
			"platform/stage/eu-central-1/s3":  {},
			"platform/stage/eu-central-1/eks": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/rds": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/app": {
				"platform/stage/eu-central-1/eks",
				"platform/stage/eu-central-1/rds",
				"platform/stage/eu-central-1/s3",
			},
		}).
		generate()

	assertPipeline(t, pipeline).
		jobStageBefore("apply-platform-stage-eu-central-1-vpc", "apply-platform-stage-eu-central-1-eks").
		jobStageBefore("apply-platform-stage-eu-central-1-eks", "apply-platform-stage-eu-central-1-app")
}

func TestPipelineGeneration_EmptyModules(t *testing.T) {
	modules := createTestModules()
	pipeline := newGeneratorScenario(t).
		withConfig(func(cfg *Config) { cfg.PlanEnabled = true }).
		withModules(modules...).
		withDependencies(map[string][]string{
			"platform/stage/eu-central-1/vpc": {},
			"platform/stage/eu-central-1/s3":  {},
			"platform/stage/eu-central-1/eks": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/rds": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/app": {
				"platform/stage/eu-central-1/eks",
				"platform/stage/eu-central-1/rds",
				"platform/stage/eu-central-1/s3",
			},
		}).
		withTargets(nil...).
		generate()

	assertPipeline(t, pipeline).jobCount(10)
}

func TestPipelineGeneration_SingleModule(t *testing.T) {
	modules := createTestModules()
	pipeline := newGeneratorScenario(t).
		withConfig(func(cfg *Config) { cfg.PlanEnabled = true }).
		withModules(modules...).
		withDependencies(map[string][]string{
			"platform/stage/eu-central-1/vpc": {},
			"platform/stage/eu-central-1/s3":  {},
			"platform/stage/eu-central-1/eks": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/rds": {"platform/stage/eu-central-1/vpc"},
			"platform/stage/eu-central-1/app": {
				"platform/stage/eu-central-1/eks",
				"platform/stage/eu-central-1/rds",
				"platform/stage/eu-central-1/s3",
			},
		}).
		withTargets(modules[0]).
		generate()

	assertPipeline(t, pipeline).
		jobCount(2).
		job("plan-platform-stage-eu-central-1-vpc").
		noNeeds()
}
