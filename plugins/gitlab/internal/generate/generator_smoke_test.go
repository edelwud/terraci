package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
)

// testCfg is a local wrapper used by tests to hold both gitlab and contributed pipeline data.
type testCfg struct {
	GitLab        *Config
	Execution     execution.Config
	Contributions []*pipeline.Contribution
}

// createTestConfig creates a test configuration with default values
func createTestConfig() *testCfg {
	return &testCfg{
		GitLab: &Config{
			Image: Image{
				Name: "hashicorp/terraform:1.6",
			},
		},
		Execution: execution.Config{
			Binary:      "terraform",
			InitEnabled: true,
			PlanEnabled: true,
			PlanMode:    execution.PlanModeStandard,
			Parallelism: 4,
		},
	}
}

func TestNewGenerator(t *testing.T) {
	cfg := createTestConfig()
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	modules := []*discovery.Module{module}
	depGraph := graph.NewDependencyGraph()
	depGraph.AddNode(module)

	gen := newTestGenerator(t, cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)

	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if gen.settings.config != cfg.GitLab {
		t.Error("config not set correctly")
	}
	if gen.ir == nil || len(gen.ir.Levels) != 1 || len(gen.ir.Levels[0].Modules) != 1 {
		t.Errorf("expected 1 module in IR, got ir=%v", gen.ir)
	}
}

func TestGenerator_Generate_SingleModule(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	p := newGeneratorScenario(t).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertPipeline(t, p).
		stageCount(2).
		jobCount(2).
		hasJob("plan-platform-stage-eu-central-1-vpc").
		hasJob("apply-platform-stage-eu-central-1-vpc")
}

func TestGenerator_Generate_WithDependencies(t *testing.T) {
	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	p := newGeneratorScenario(t).
		withModules(vpc, eks).
		withDependencies(map[string][]string{
			vpc.ID(): {},
			eks.ID(): {vpc.ID()},
		}).
		generate()

	assertPipeline(t, p).
		stageCount(4).
		job("apply-platform-stage-eu-central-1-eks").
		hasNeed("apply-platform-stage-eu-central-1-vpc")
}

func TestGenerator_Generate_PlanOnly(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	p := newGeneratorScenario(t).
		withConfig(func(cfg *Config) {
			cfg.PlanOnly = true
		}).
		withExecution(func(cfg *execution.Config) {
			cfg.PlanEnabled = true
		}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertPipeline(t, p).
		stageCount(1).
		jobCount(1).
		hasJob("plan-platform-stage-eu-central-1-vpc").
		noJob("apply-platform-stage-eu-central-1-vpc")
}

func TestGenerator_Generate_PlanOnlyWithDependencies(t *testing.T) {
	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	p := newGeneratorScenario(t).
		withConfig(func(cfg *Config) {
			cfg.PlanOnly = true
		}).
		withExecution(func(cfg *execution.Config) {
			cfg.PlanEnabled = true
		}).
		withModules(vpc, eks).
		withDependencies(map[string][]string{
			vpc.ID(): {},
			eks.ID(): {vpc.ID()},
		}).
		generate()

	assertPipeline(t, p).
		job("plan-platform-stage-eu-central-1-eks").
		hasNeed("plan-platform-stage-eu-central-1-vpc").
		noNeedWithPrefix("apply-")
}

func TestGenerator_Generate_AutoApprove(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	p := newGeneratorScenario(t).
		withConfig(func(cfg *Config) { cfg.AutoApprove = true }).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertPipeline(t, p).
		job("apply-platform-stage-eu-central-1-vpc").
		notManual()
}

func TestGenerator_Generate_ManualApprove(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	p := newGeneratorScenario(t).
		withConfig(func(cfg *Config) { cfg.AutoApprove = false }).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertPipeline(t, p).
		job("apply-platform-stage-eu-central-1-vpc").
		when("manual")
}

func TestGenerator_Generate_CustomStagesPrefix(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	p := newGeneratorScenario(t).
		withConfig(func(cfg *Config) { cfg.StagesPrefix = "terraform" }).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertPipeline(t, p).stagesHavePrefix("terraform-")
}

func TestGenerator_Generate_ExecutionBinary(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	p := newGeneratorScenario(t).
		withExecution(func(cfg *execution.Config) { cfg.Binary = "tofu" }).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertPipeline(t, p).variable("TERRAFORM_BINARY", "tofu")
}

func TestGenerator_Generate_JobVariables(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	p := newGeneratorScenario(t).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	job := assertPipeline(t, p).job("plan-platform-stage-eu-central-1-vpc")
	job.variable("TF_MODULE_PATH", "platform/stage/eu-central-1/vpc")
	job.variable("TF_SERVICE", "platform")
	job.variable("TF_ENVIRONMENT", "stage")
	job.variable("TF_REGION", "eu-central-1")
	job.variable("TF_MODULE", "vpc")
}

func TestGenerator_Generate_ResourceGroup(t *testing.T) {
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	p := newGeneratorScenario(t).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	expectedResourceGroup := "platform/stage/eu-central-1/vpc"
	assertPipeline(t, p).
		job("plan-platform-stage-eu-central-1-vpc").
		resourceGroup(expectedResourceGroup)
	assertPipeline(t, p).
		job("apply-platform-stage-eu-central-1-vpc").
		resourceGroup(expectedResourceGroup)
}

func TestGenerator_DryRun(t *testing.T) {
	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
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
		Stages:          4,
		ExecutionLevels: 2,
	})
}
