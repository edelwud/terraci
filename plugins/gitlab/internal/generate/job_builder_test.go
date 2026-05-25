package generate

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/pipelinetest"
	"github.com/edelwud/terraci/pkg/workflow"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	"github.com/edelwud/terraci/plugins/gitlab/internal/domain"
)

func testExecutionConfig() execution.Config {
	return execution.Config{Binary: "terraform", InitEnabled: true, PlanEnabled: true, PlanMode: execution.PlanModeStandard, Parallelism: 4}
}

func noJobConfig(_ *domain.Job, _ configpkg.JobOverwriteType) error { return nil }

func TestJobBuilderRenderJobBuildsModuleDefaults(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	builder := newJobBuilder(
		newSettings(&configpkg.Config{CacheEnabled: true}, testExecutionConfig()),
		map[string]string{"plan-platform-stage-eu-central-1-vpc": "deploy-0"},
		noJobConfig,
	)

	plan := pipelinetest.MustJobByKind(t, pipelinetest.MustSingleModuleIR(t, module), pipeline.JobKindPlan)
	job, err := builder.renderJob(&plan)
	if err != nil {
		t.Fatalf("renderJob() error = %v", err)
	}
	if job.Stage != "deploy-0" {
		t.Fatalf("Stage = %q", job.Stage)
	}
	if len(job.Script) == 0 || !strings.Contains(strings.Join(job.Script, "\n"), "plan") {
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
	if len(job.Needs) != 0 {
		t.Fatalf("Needs = %#v, want none", job.Needs)
	}
}

func TestJobBuilderCacheSupportsAdvancedOptions(t *testing.T) {
	t.Parallel()

	enabled := true
	builder := newJobBuilder(
		newSettings(&configpkg.Config{
			CacheEnabled: true,
			Cache: &configpkg.CacheConfig{
				Enabled: &enabled,
				Key:     "terraform-{service}-{environment}-{module}",
				Paths:   []string{"{module_path}/.terraform/", "{module_path}/.terraform.lock.hcl"},
				Policy:  "pull-push",
			},
		}, testExecutionConfig()),
		nil,
		noJobConfig,
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
}

func TestJobBuilderContributedJobOverwriteByName(t *testing.T) {
	t.Parallel()

	cfg := &configpkg.Config{
		JobDefaults: &configpkg.JobDefaults{Tags: []string{"default-runner"}},
		Overwrites: []configpkg.JobOverwrite{{
			Type:  "cost-estimation",
			Tags:  []string{"cost-runner"},
			Image: &configpkg.Image{Name: "cost-image:1.0"},
		}},
	}
	s := newSettings(cfg, testExecutionConfig())
	builder := newJobBuilder(
		s,
		map[string]string{"cost-estimation": "deploy-1"},
		func(job *domain.Job, jt configpkg.JobOverwriteType) error {
			return applyResolvedJobConfig(s, job, jt)
		},
	)

	command := pipelinetest.MustCommandJob(t, pipeline.ContributedJobOptions{Name: "cost-estimation", Commands: []string{"terraci cost"}})
	job, err := builder.renderJob(&command)
	if err != nil {
		t.Fatalf("renderJob() error = %v", err)
	}
	if len(job.Tags) != 1 || job.Tags[0] != "cost-runner" {
		t.Fatalf("Tags = %v, want [cost-runner]", job.Tags)
	}
	if job.Image == nil || job.Image.Name != "cost-image:1.0" {
		t.Fatalf("Image = %v, want cost-image:1.0", job.Image)
	}
}

func TestJobBuilderContributedJobUsesOptionalNeeds(t *testing.T) {
	t.Parallel()

	builder := newJobBuilder(
		newSettings(&configpkg.Config{}, testExecutionConfig()),
		map[string]string{"summary": "deploy-2"},
		noJobConfig,
	)

	producer, err := pipeline.NewPluginCommandJob(pipeline.PluginCommandJobOptions{
		Name:     "apply-a",
		Commands: []string{"apply-a"},
		Produces: []pipeline.ResourceSpec{
			pipeline.PluginResource(pipeline.ResourceKindPluginReport, "apply-a", ".terraci/apply-a.json"),
		},
	})
	if err != nil {
		t.Fatalf("NewPluginCommandJob(producer) error = %v", err)
	}
	consumer, err := pipeline.NewPluginCommandJob(pipeline.PluginCommandJobOptions{
		Name:         "summary",
		Commands:     []string{"terraci summary"},
		Consumes:     []pipeline.ResourceRequest{pipeline.PluginProducerResource(pipeline.ResourceKindPluginReport, "apply-a", true)},
		AllowFailure: true,
	})
	if err != nil {
		t.Fatalf("NewPluginCommandJob(consumer) error = %v", err)
	}
	contribution, err := pipeline.NewContribution(producer, consumer)
	if err != nil {
		t.Fatalf("NewContribution() error = %v", err)
	}
	plan, err := pipeline.BuildProjectIR(pipeline.ProjectIRRequest{Contributions: []*pipeline.Contribution{contribution}, Project: emptyProject()})
	if err != nil {
		t.Fatalf("BuildProjectIR() error = %v", err)
	}
	summary := findJobByName(t, plan, "summary")
	job, err := builder.renderJob(&summary)
	if err != nil {
		t.Fatalf("renderJob() error = %v", err)
	}
	if job.Stage != "deploy-2" {
		t.Fatalf("Stage = %q", job.Stage)
	}
	if len(job.Needs) != 1 || !job.Needs[0].Optional {
		t.Fatalf("Needs = %#v", job.Needs)
	}
	if job.Needs[0].Artifacts == nil || !*job.Needs[0].Artifacts {
		t.Fatalf("Needs[0].Artifacts = %#v, want true", job.Needs[0].Artifacts)
	}
	if len(job.Script) != 1 || job.Script[0] != "terraci summary || true" {
		t.Fatalf("Script = %#v", job.Script)
	}
}

func emptyProject() *workflow.ProjectResult {
	return &workflow.ProjectResult{
		Workflow: &workflow.Result{
			Filtered: workflow.NewModuleSet(nil),
			Graph:    graph.NewDependencyGraph(),
		},
		Targets: []*discovery.Module{},
	}
}

func findJobByName(tb testing.TB, ir *pipeline.IR, name string) pipeline.Job {
	tb.Helper()
	jobs := ir.Jobs()
	for i := range jobs {
		if jobs[i].Name() == name {
			return jobs[i]
		}
	}
	tb.Fatalf("job %q not found", name)
	var zero pipeline.Job
	return zero
}
