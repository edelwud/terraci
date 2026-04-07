package generate

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci/citest"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

func TestGenerate_SingleModule(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		jobCount(2).
		hasJob("plan-platform-stage-eu-central-1-vpc").
		hasJob("apply-platform-stage-eu-central-1-vpc")

	assertWorkflow(t, workflow).
		job("plan-platform-stage-eu-central-1-vpc").
		stepNamed("Checkout").
		stepUses("actions/checkout@v4").
		stepRunContains("${TERRAFORM_BINARY} init").
		stepRunContains("${TERRAFORM_BINARY} plan").
		stepUses("actions/upload-artifact@v4")
}

func TestGenerate_WithDependencies(t *testing.T) {
	vpc := createTestModule("platform", "stage", "eu-central-1", "vpc")
	eks := createTestModule("platform", "stage", "eu-central-1", "eks")
	workflow := newGeneratorScenario(t).
		withModules(vpc, eks).
		withDependencies(map[string][]string{
			vpc.ID(): {},
			eks.ID(): {vpc.ID()},
		}).
		generate()

	assertWorkflow(t, workflow).
		job("apply-platform-stage-eu-central-1-eks").
		hasNeed("apply-platform-stage-eu-central-1-vpc")
}

func TestGenerate_PlanOnly(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) {
			cfg.PlanOnly = true
			cfg.PlanEnabled = true
		}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		jobCount(1).
		hasJob("plan-platform-stage-eu-central-1-vpc").
		noJob("apply-platform-stage-eu-central-1-vpc")
}

func TestGenerate_PlanOnlyWithDeps(t *testing.T) {
	vpc := createTestModule("platform", "stage", "eu-central-1", "vpc")
	eks := createTestModule("platform", "stage", "eu-central-1", "eks")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) {
			cfg.PlanOnly = true
			cfg.PlanEnabled = true
		}).
		withModules(vpc, eks).
		withDependencies(map[string][]string{
			vpc.ID(): {},
			eks.ID(): {vpc.ID()},
		}).
		generate()

	assertWorkflow(t, workflow).
		job("plan-platform-stage-eu-central-1-eks").
		hasNeed("plan-platform-stage-eu-central-1-vpc").
		noNeedWithPrefix("apply-")
}

func TestGenerate_AutoApprove(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) { cfg.AutoApprove = true }).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		job("apply-platform-stage-eu-central-1-vpc").
		noEnvironment()
}

func TestGenerate_ManualApprove(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) { cfg.AutoApprove = false }).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		job("apply-platform-stage-eu-central-1-vpc").
		environment("production")
}

func TestGenerate_CustomBinary(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) { cfg.TerraformBinary = "tofu" }).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).env("TERRAFORM_BINARY", "tofu")
}

func TestGenerate_WithContainer(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) {
			cfg.Container = &configpkg.Image{Name: "hashicorp/terraform:1.6"}
		}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		job("plan-platform-stage-eu-central-1-vpc").
		containerImage("hashicorp/terraform:1.6")
}

func TestGenerate_StepsBefore(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) {
			cfg.JobDefaults = &configpkg.JobDefaults{
				StepsBefore: []configpkg.ConfigStep{
					{Name: "Setup AWS credentials", Uses: "aws-actions/configure-aws-credentials@v4"},
				},
			}
		}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	job := workflow.Jobs["plan-platform-stage-eu-central-1-vpc"]
	if job == nil {
		t.Fatal("plan job not found")
	}

	setupIdx := -1
	planIdx := -1
	for i, step := range job.Steps {
		if step.Name == "Setup AWS credentials" {
			setupIdx = i
		}
		if strings.HasPrefix(step.Name, "Plan ") {
			planIdx = i
		}
	}
	if setupIdx == -1 {
		t.Fatal("steps_before step not found in plan job")
	}
	if planIdx == -1 {
		t.Fatal("plan step not found in plan job")
	}
	if setupIdx >= planIdx {
		t.Errorf("steps_before should appear before plan step: setup=%d, plan=%d", setupIdx, planIdx)
	}
}

func TestDryRun(t *testing.T) {
	vpc := createTestModule("platform", "stage", "eu-central-1", "vpc")
	eks := createTestModule("platform", "stage", "eu-central-1", "eks")
	result := newGeneratorScenario(t).
		withModules(vpc, eks).
		withDependencies(map[string][]string{
			vpc.ID(): {},
			eks.ID(): {vpc.ID()},
		}).
		dryRun()

	citest.AssertDryRun(t, result, citest.DryRunExpectation{
		TotalModules:    2,
		AffectedModules: 2,
		Jobs:            4,
		Stages:          2,
		ExecutionLevels: 2,
	})
}
