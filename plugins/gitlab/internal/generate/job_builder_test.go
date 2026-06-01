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
)

func testExecutionConfig() execution.Config {
	return execution.Config{Binary: "terraform", InitEnabled: true, Parallelism: 4}
}

func TestJobBuilderRenderJobBuildsModuleDefaults(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	builder := newJobBuilder(
		newSettings(&configpkg.Config{}, testExecutionConfig()),
		map[string]string{"plan-platform-stage-eu-central-1-vpc": "deploy-0"},
	)

	plan := pipelinetest.MustJobByKind(t, pipelinetest.MustSingleModuleIR(t, module), pipeline.JobKindPlan)
	job, err := builder.renderJob(plan)
	if err != nil {
		t.Fatalf("renderJob() error = %v", err)
	}
	if job.Stage() != "deploy-0" {
		t.Fatalf("Stage = %q", job.Stage())
	}
	if script := job.Script(); len(script) == 0 || !strings.Contains(strings.Join(script, "\n"), "plan") {
		t.Fatalf("Script = %#v", script)
	}
	if cache := job.Cache(); cache == nil || cache.Key == "" {
		t.Fatal("expected cache to be populated")
	}
	if job.ResourceGroup() != module.ID() {
		t.Fatalf("ResourceGroup = %q", job.ResourceGroup())
	}
	if artifacts := job.Artifacts(); artifacts == nil || len(artifacts.Paths) != 1 {
		t.Fatal("expected default artifacts")
	}
	if needs := job.Needs(); len(needs) != 0 {
		t.Fatalf("Needs = %#v, want none", needs)
	}
}

func TestJobBuilderCacheSupportsAdvancedOptions(t *testing.T) {
	t.Parallel()

	enabled := true
	builder := newJobBuilder(
		newSettings(&configpkg.Config{
			Cache: &configpkg.CacheConfig{
				Enabled: &enabled,
				Key:     "terraform-{service}-{environment}-{module}",
				Paths:   []string{"{module_path}/.terraform/", "{module_path}/.terraform.lock.hcl"},
				Policy:  "pull-push",
			},
		}, testExecutionConfig()),
		nil,
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
	)

	command := pipelinetest.MustCommandJob(t, pipeline.ContributedJobOptions{Name: "cost-estimation", Commands: []string{"terraci cost"}})
	job, err := builder.renderJob(command)
	if err != nil {
		t.Fatalf("renderJob() error = %v", err)
	}
	if tags := job.Tags(); len(tags) != 1 || tags[0] != "cost-runner" {
		t.Fatalf("Tags = %v, want [cost-runner]", tags)
	}
	if image := job.Image(); image == nil || image.Name != "cost-image:1.0" {
		t.Fatalf("Image = %v, want cost-image:1.0", image)
	}
}

func TestJobBuilderContributedJobUsesOptionalNeeds(t *testing.T) {
	t.Parallel()

	builder := newJobBuilder(
		newSettings(&configpkg.Config{}, testExecutionConfig()),
		map[string]string{"summary": "deploy-2"},
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
	job, err := builder.renderJob(summary)
	if err != nil {
		t.Fatalf("renderJob() error = %v", err)
	}
	if job.Stage() != "deploy-2" {
		t.Fatalf("Stage = %q", job.Stage())
	}
	needs := job.Needs()
	if len(needs) != 1 || !needs[0].Optional {
		t.Fatalf("Needs = %#v", needs)
	}
	if needs[0].Artifacts == nil || !*needs[0].Artifacts {
		t.Fatalf("Needs[0].Artifacts = %#v, want true", needs[0].Artifacts)
	}
	if script := job.Script(); len(script) != 1 || script[0] != "terraci summary || true" {
		t.Fatalf("Script = %#v", script)
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
