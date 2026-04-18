package generate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	"github.com/edelwud/terraci/plugins/gitlab/internal/domain"
)

func testExecutionConfig() execution.Config {
	return execution.Config{
		Binary:      "terraform",
		InitEnabled: true,
		PlanEnabled: true,
		PlanMode:    execution.PlanModeStandard,
		Parallelism: 4,
	}
}

func TestJobBuilderPlanJobBuildsExpectedDefaults(t *testing.T) {
	builder := newJobBuilder(
		newSettings(&configpkg.Config{CacheEnabled: true}, testExecutionConfig()),
		contributionIndex{},
		func(_ *domain.Job, _ configpkg.JobOverwriteType) {},
	)

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	irJob := &pipeline.Job{
		Operation: pipeline.Operation{
			Type:     pipeline.OperationTypeCommands,
			Commands: []string{"terraform plan"},
		},
		Dependencies:  []string{"apply-platform-stage-eu-central-1-base"},
		ArtifactPaths: []string{"plan.json"},
		Env:           map[string]string{"TF_VAR_env": "stage"},
		Steps: []pipeline.Step{
			{Phase: pipeline.PhasePrePlan, Command: "echo before"},
			{Phase: pipeline.PhasePostPlan, Command: "echo after"},
		},
	}

	job := builder.planJob(irJob, module, 2, "deploy")
	if job.Stage != "deploy-plan-2" {
		t.Fatalf("Stage = %q", job.Stage)
	}
	if len(job.Script) != 3 || job.Script[0] != "echo before" || job.Script[2] != "echo after" {
		t.Fatalf("Script = %#v", job.Script)
	}
	if job.Cache == nil || job.Cache.Key == "" {
		t.Fatal("expected cache to be populated")
	}
	if job.ResourceGroup != module.ID() {
		t.Fatalf("ResourceGroup = %q", job.ResourceGroup)
	}
	if job.Artifacts == nil || len(job.Artifacts.Paths) != 1 {
		t.Fatal("expected default artifacts")
	}
	if len(job.Needs) != 1 || job.Needs[0].Optional {
		t.Fatalf("Needs = %#v", job.Needs)
	}
}

func TestJobBuilderApplyJobHonorsAutoApprove(t *testing.T) {
	builder := newJobBuilder(
		newSettings(&configpkg.Config{AutoApprove: false}, testExecutionConfig()),
		contributionIndex{},
		func(_ *domain.Job, _ configpkg.JobOverwriteType) {},
	)

	job := builder.applyJob(&pipeline.Job{}, discovery.TestModule("platform", "stage", "eu-central-1", "vpc"), 0, "deploy")
	if job.When != WhenManual {
		t.Fatalf("When = %q", job.When)
	}
}

func TestJobBuilderCacheSupportsAdvancedOptions(t *testing.T) {
	enabled := true
	builder := newJobBuilder(
		newSettings(&configpkg.Config{
			CacheEnabled: true,
			Cache: &configpkg.CacheConfig{
				Enabled: &enabled,
				Key:     "terraform-{service}-{environment}-{module}",
				Paths: []string{
					"{module_path}/.terraform/",
					"{module_path}/.terraform.lock.hcl",
				},
				Policy: "pull-push",
			},
		}, testExecutionConfig()),
		contributionIndex{},
		func(_ *domain.Job, _ configpkg.JobOverwriteType) {},
	)

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")

	cache := builder.cache(module)
	if cache == nil {
		t.Fatal("expected cache to be populated")
	}
	if cache.Key != "terraform-platform-stage-vpc" {
		t.Fatalf("Key = %q", cache.Key)
	}
	if cache.Policy != "pull-push" {
		t.Fatalf("Policy = %q", cache.Policy)
	}
	if len(cache.Paths) != 2 {
		t.Fatalf("Paths = %#v", cache.Paths)
	}
	if cache.Paths[0] != "platform/stage/eu-central-1/vpc/.terraform/" {
		t.Fatalf("Paths[0] = %q", cache.Paths[0])
	}
	if cache.Paths[1] != "platform/stage/eu-central-1/vpc/.terraform.lock.hcl" {
		t.Fatalf("Paths[1] = %q", cache.Paths[1])
	}
}

func TestJobBuilderCacheConfigCanDisableLegacyCache(t *testing.T) {
	enabled := false
	builder := newJobBuilder(
		newSettings(&configpkg.Config{
			CacheEnabled: true,
			Cache: &configpkg.CacheConfig{
				Enabled: &enabled,
			},
		}, testExecutionConfig()),
		contributionIndex{},
		func(_ *domain.Job, _ configpkg.JobOverwriteType) {},
	)

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	if cache := builder.cache(module); cache != nil {
		t.Fatalf("cache = %#v, want nil", cache)
	}
}

func TestJobBuilderContributedJobInheritsJobDefaults(t *testing.T) {
	cfg := &configpkg.Config{
		JobDefaults: &configpkg.JobDefaults{
			Image:        &configpkg.Image{Name: "custom:latest"},
			Tags:         []string{"k8s"},
			BeforeScript: []string{"echo setup"},
		},
	}
	s := newSettings(cfg, testExecutionConfig())
	builder := newJobBuilder(
		s,
		contributionIndex{
			hasJobs:    true,
			stageByJob: map[string]string{"cost-estimation": "post-plan"},
		},
		func(job *domain.Job, jt configpkg.JobOverwriteType) {
			applyResolvedJobConfig(s, job, jt)
		},
	)

	job := builder.contributedJob(&pipeline.Job{
		Name: "cost-estimation",
		Operation: pipeline.Operation{
			Type:     pipeline.OperationTypeCommands,
			Commands: []string{"terraci cost"},
		},
	})

	if job.Image == nil || job.Image.Name != "custom:latest" {
		t.Fatalf("Image = %v, want custom:latest", job.Image)
	}
	if len(job.Tags) != 1 || job.Tags[0] != "k8s" {
		t.Fatalf("Tags = %v, want [k8s]", job.Tags)
	}
	if len(job.BeforeScript) != 1 || job.BeforeScript[0] != "echo setup" {
		t.Fatalf("BeforeScript = %v, want [echo setup]", job.BeforeScript)
	}
}

func TestJobBuilderContributedJobOverwriteByName(t *testing.T) {
	cfg := &configpkg.Config{
		JobDefaults: &configpkg.JobDefaults{
			Tags: []string{"default-runner"},
		},
		Overwrites: []configpkg.JobOverwrite{{
			Type:  "cost-estimation",
			Tags:  []string{"cost-runner"},
			Image: &configpkg.Image{Name: "cost-image:1.0"},
		}},
	}
	s := newSettings(cfg, testExecutionConfig())
	builder := newJobBuilder(
		s,
		contributionIndex{
			hasJobs:    true,
			stageByJob: map[string]string{"cost-estimation": "post-plan"},
		},
		func(job *domain.Job, jt configpkg.JobOverwriteType) {
			applyResolvedJobConfig(s, job, jt)
		},
	)

	job := builder.contributedJob(&pipeline.Job{
		Name: "cost-estimation",
		Operation: pipeline.Operation{
			Type:     pipeline.OperationTypeCommands,
			Commands: []string{"terraci cost"},
		},
	})

	// Overwrite should win over defaults
	if len(job.Tags) != 1 || job.Tags[0] != "cost-runner" {
		t.Fatalf("Tags = %v, want [cost-runner]", job.Tags)
	}
	if job.Image == nil || job.Image.Name != "cost-image:1.0" {
		t.Fatalf("Image = %v, want cost-image:1.0", job.Image)
	}
}

func TestJobBuilderContributedJobUsesOptionalNeedsAndSummaryOverrides(t *testing.T) {
	builder := newJobBuilder(
		newSettings(&configpkg.Config{
			MR: &configpkg.MRConfig{
				SummaryJob: &configpkg.SummaryJobConfig{
					Tags: []string{"docker"},
				},
			},
		}, testExecutionConfig()),
		contributionIndex{
			hasJobs:    true,
			stageByJob: map[string]string{"summary": "finalize"},
		},
		func(_ *domain.Job, _ configpkg.JobOverwriteType) {},
	)

	job := builder.contributedJob(&pipeline.Job{
		Name:         "summary",
		Dependencies: []string{"apply-a"},
		AllowFailure: true,
		Operation: pipeline.Operation{
			Type:     pipeline.OperationTypeCommands,
			Commands: []string{"terraci summary"},
		},
	})
	builder.applySummaryOverrides(job)

	if job.Stage != "finalize" {
		t.Fatalf("Stage = %q", job.Stage)
	}
	if len(job.Needs) != 1 || !job.Needs[0].Optional {
		t.Fatalf("Needs = %#v", job.Needs)
	}
	if len(job.Script) != 1 || job.Script[0] != "terraci summary || true" {
		t.Fatalf("Script = %#v", job.Script)
	}
	if len(job.Tags) != 1 || job.Tags[0] != "docker" {
		t.Fatalf("Tags = %#v", job.Tags)
	}
	if len(job.Rules) != 1 || job.Rules[0].If != "$CI_MERGE_REQUEST_IID" {
		t.Fatalf("Rules = %#v", job.Rules)
	}
}
