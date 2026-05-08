package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

func TestGenerate_WithSummaryContribution(t *testing.T) {
	vpc := createTestModule("platform", "stage", "eu-central-1", "vpc")
	eks := createTestModule("platform", "stage", "eu-central-1", "eks")
	workflow := newGeneratorScenario(t).
		withContributions([]*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{{
				Name:     "terraci-summary",
				Commands: []string{"terraci summary"},
				Consumes: []pipeline.ResourceRequest{
					pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
				},
				AllowFailure: false,
			}},
		}}).
		withModules(vpc, eks).
		withDependencies(map[string][]string{
			vpc.ID(): {},
			eks.ID(): {vpc.ID()},
		}).
		generate()

	assertWorkflow(t, workflow).hasJob("terraci-summary")
	assertWorkflow(t, workflow).
		job("terraci-summary").
		hasNeed("plan-platform-stage-eu-central-1-vpc").
		hasNeed("plan-platform-stage-eu-central-1-eks")
}

func TestGenerate_ContributedJobInheritsJobDefaults(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) {
			cfg.JobDefaults = &configpkg.JobDefaults{
				Container:   &configpkg.Image{Name: "custom:latest"},
				StepsBefore: []configpkg.ConfigStep{{Name: "Setup", Run: "echo setup"}},
				StepsAfter:  []configpkg.ConfigStep{{Name: "Cleanup", Run: "echo cleanup"}},
			}
		}).
		withContributions([]*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{{
				Name:     "cost-estimation",
				Commands: []string{"terraci cost"},
				Consumes: []pipeline.ResourceRequest{
					pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
				},
				AllowFailure: true,
			}},
		}}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		hasJob("cost-estimation").
		job("cost-estimation").
		containerImage("custom:latest").
		stepNamed("Setup").
		stepNamed("Cleanup")
}

func TestGenerate_ContributedJobOverwriteByName(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) {
			cfg.JobDefaults = &configpkg.JobDefaults{
				Container: &configpkg.Image{Name: "default:latest"},
			}
			cfg.Overwrites = []configpkg.JobOverwrite{{
				Type:      "cost-estimation",
				Container: &configpkg.Image{Name: "cost-specific:1.0"},
				RunsOn:    "cost-runner",
			}}
		}).
		withContributions([]*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{{
				Name:     "cost-estimation",
				Commands: []string{"terraci cost"},
				Consumes: []pipeline.ResourceRequest{
					pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
				},
				AllowFailure: true,
			}},
		}}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		hasJob("cost-estimation").
		job("cost-estimation").
		containerImage("cost-specific:1.0")

	job := workflow.Jobs["cost-estimation"]
	if job.RunsOn != "cost-runner" {
		t.Fatalf("RunsOn = %q, want cost-runner", job.RunsOn)
	}
}

func TestGenerate_PlanAndApplyJobOverwritesUseResolvedProfile(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) {
			cfg.JobDefaults = &configpkg.JobDefaults{
				RunsOn:      "default-runner",
				Container:   &configpkg.Image{Name: "default:latest"},
				Env:         map[string]string{"DEFAULT": "true", "SHARED": "default"},
				StepsBefore: []configpkg.ConfigStep{{Name: "Default setup", Run: "echo default setup"}},
				StepsAfter:  []configpkg.ConfigStep{{Name: "Default cleanup", Run: "echo default cleanup"}},
			}
			cfg.Overwrites = []configpkg.JobOverwrite{
				{
					Type:        configpkg.OverwriteTypePlan,
					RunsOn:      "plan-runner",
					Container:   &configpkg.Image{Name: "plan:latest"},
					Env:         map[string]string{"PLAN": "true", "SHARED": "plan"},
					StepsBefore: []configpkg.ConfigStep{{Name: "Plan setup", Run: "echo plan setup"}},
					StepsAfter:  []configpkg.ConfigStep{{Name: "Plan cleanup", Run: "echo plan cleanup"}},
				},
				{
					Type:        configpkg.OverwriteTypeApply,
					RunsOn:      "apply-runner",
					Env:         map[string]string{"APPLY": "true"},
					StepsBefore: []configpkg.ConfigStep{{Name: "Apply setup", Run: "echo apply setup"}},
				},
			}
		}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		job("plan-platform-stage-eu-central-1-vpc").
		runsOn("plan-runner").
		containerImage("plan:latest").
		env("DEFAULT", "true").
		env("SHARED", "plan").
		env("PLAN", "true").
		env("TF_MODULE", "vpc").
		stepNamed("Default setup").
		stepNamed("Plan setup").
		stepNamed("Default cleanup").
		stepNamed("Plan cleanup")

	assertWorkflow(t, workflow).
		job("apply-platform-stage-eu-central-1-vpc").
		runsOn("apply-runner").
		containerImage("default:latest").
		env("DEFAULT", "true").
		env("SHARED", "default").
		env("APPLY", "true").
		env("TF_MODULE", "vpc").
		stepNamed("Default setup").
		stepNamed("Apply setup").
		stepNamed("Default cleanup")
}

func TestGenerate_ContributedJobAppliesAllMatchingOverwritesInOrder(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withConfig(func(cfg *configpkg.Config) {
			cfg.JobDefaults = &configpkg.JobDefaults{
				RunsOn:      "default-runner",
				Container:   &configpkg.Image{Name: "default:latest"},
				Env:         map[string]string{"DEFAULT": "true", "SHARED": "default"},
				StepsBefore: []configpkg.ConfigStep{{Name: "Default setup", Run: "echo default setup"}},
			}
			cfg.Overwrites = []configpkg.JobOverwrite{
				{
					Type:        "cost-estimation",
					RunsOn:      "first-runner",
					Container:   &configpkg.Image{Name: "first:latest"},
					Env:         map[string]string{"FIRST": "true", "SHARED": "first"},
					StepsBefore: []configpkg.ConfigStep{{Name: "First setup", Run: "echo first setup"}},
				},
				{
					Type:       "cost-estimation",
					RunsOn:     "second-runner",
					Env:        map[string]string{"SECOND": "true", "SHARED": "second"},
					StepsAfter: []configpkg.ConfigStep{{Name: "Second cleanup", Run: "echo second cleanup"}},
				},
			}
		}).
		withContributions([]*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{{
				Name:     "cost-estimation",
				Commands: []string{"terraci cost"},
				Consumes: []pipeline.ResourceRequest{
					pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
				},
				AllowFailure: true,
			}},
		}}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		job("cost-estimation").
		runsOn("second-runner").
		containerImage("first:latest").
		env("DEFAULT", "true").
		env("FIRST", "true").
		env("SECOND", "true").
		env("SHARED", "second").
		stepNamed("Default setup").
		stepNamed("First setup").
		stepNamed("Second cleanup")
}

func TestGenerate_WithPolicy(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	workflow := newGeneratorScenario(t).
		withContributions([]*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{{
				Name:     "policy-check",
				Commands: []string{"terraci policy pull", "terraci policy check"},
				Consumes: []pipeline.ResourceRequest{
					pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
				},
				Produces: []pipeline.ResourceSpec{
					pipeline.PluginResource(pipeline.ResourceKindPluginResult, "policy", ".terraci/policy-results.json"),
					pipeline.PluginResource(pipeline.ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
				},
				AllowFailure: false,
			}},
		}}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		hasJob("policy-check").
		job("policy-check").
		hasNeed("plan-platform-stage-eu-central-1-vpc")
}

func TestGenerate_ArtifactRestoreContract(t *testing.T) {
	module := createTestModule("platform", "stage", "eu-central-1", "vpc")
	planName := "plan-platform-stage-eu-central-1-vpc"
	planArtifact := pipeline.PlanArtifactName(planName)
	resultArtifact := pipeline.ResultArtifact("cost-estimation", ".terraci/cost-results.json", ".terraci/cost-report.json")

	workflow := newGeneratorScenario(t).
		withContributions([]*pipeline.Contribution{{
			Jobs: []pipeline.ContributedJob{{
				Name:     "cost-estimation",
				Commands: []string{"terraci cost"},
				Consumes: []pipeline.ResourceRequest{
					pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
				},
				Produces: []pipeline.ResourceSpec{
					pipeline.PluginResource(pipeline.ResourceKindPluginResult, "cost", ".terraci/cost-results.json"),
					pipeline.PluginResource(pipeline.ResourceKindPluginReport, "cost", ".terraci/cost-report.json"),
				},
				AllowFailure: true,
			}},
		}}).
		withModules(module).
		withDependencies(map[string][]string{module.ID(): {}}).
		generate()

	assertWorkflow(t, workflow).
		job(planName).
		stepNamed("Stage plan artifacts").
		stepRunContains(".terraci/artifacts/"+planArtifact+"/").
		stepRunContains("plan.json").
		stepWith("Upload plan artifacts", "name", planArtifact).
		stepWith("Upload plan artifacts", "path", ".terraci/artifacts/"+planArtifact+"/").
		stepWith("Upload plan artifacts", "include-hidden-files", "true")

	assertWorkflow(t, workflow).
		job("apply-platform-stage-eu-central-1-vpc").
		stepWith("Download "+planArtifact, "name", planArtifact).
		stepWith("Download "+planArtifact, "path", ".")

	assertWorkflow(t, workflow).
		job("cost-estimation").
		stepWith("Download "+planArtifact, "name", planArtifact).
		stepWith("Download "+planArtifact, "path", ".").
		stepWith("Upload cost-estimation results", "name", resultArtifact.Name).
		stepWith("Upload cost-estimation results", "include-hidden-files", "true")
}
