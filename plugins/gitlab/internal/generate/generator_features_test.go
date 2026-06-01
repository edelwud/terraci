package generate

import (
	"slices"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
)

func testContribution(tb testing.TB, opts ...pipeline.ContributedJobOptions) *pipeline.Contribution {
	tb.Helper()
	jobs := make([]pipeline.ContributedJob, 0, len(opts))
	for _, opt := range opts {
		job, err := pipeline.NewContributedJob(opt)
		if err != nil {
			tb.Fatalf("NewContributedJob() error = %v", err)
		}
		jobs = append(jobs, job)
	}
	contribution, err := pipeline.NewContribution(jobs...)
	if err != nil {
		tb.Fatalf("NewContribution() error = %v", err)
	}
	return contribution
}

func TestGenerator_Generate_WithSummaryContribution(t *testing.T) {
	cfg := createTestConfig()
	cfg.Contributions = []*pipeline.Contribution{testContribution(t, pipeline.ContributedJobOptions{
		Name:     "terraci-summary",
		Commands: []string{"terraci summary"},
		Consumes: []pipeline.ResourceRequest{
			pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
		},
		AllowFailure: false,
	})}

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
	}

	depGraph := citest.DependencyGraph(modules, map[string][]string{
		modules[0].ID(): {},
		modules[1].ID(): {modules[0].ID()},
	})

	gen := newTestGenerator(t, cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)
	genPipeline, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	assertPipeline(t, p).
		hasJob("terraci-summary")
	summaryJob := mustJob(t, p, "terraci-summary")
	stages := p.Stages()
	if summaryJob.Stage() != stages[len(stages)-1] {
		t.Errorf("summary job stage: expected last DAG stage, got %s in %v", summaryJob.Stage(), stages)
	}
	if len(summaryJob.Needs()) != 2 {
		t.Errorf("summary job should have 2 needs, got %d", len(summaryJob.Needs()))
	}
	assertPipeline(t, p).
		job("plan-platform-stage-eu-central-1-vpc").
		artifactPathContains("plan.json")
}

func TestGenerator_Generate_WithMRIntegration_Disabled(t *testing.T) {
	cfg := createTestConfig()
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	depGraph := citest.DependencyGraph(modules, map[string][]string{
		modules[0].ID(): {},
	})

	gen := newTestGenerator(t, cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)
	genPipeline, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	assertPipeline(t, p).noJob("terraci-summary")
}

func TestGenerator_Generate_WithSecrets(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.JobDefaults = &JobDefaults{
		Secrets: map[string]CfgSecret{
			"AWS_SECRET_KEY": {
				Vault: &CfgVaultSecret{
					Shorthand: "secret/data/aws/key@production",
				},
				File: true,
			},
		},
	}

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}
	depGraph := citest.DependencyGraph(modules, map[string][]string{
		modules[0].ID(): {},
	})

	gen := newTestGenerator(t, cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)
	genPipeline, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	planJob := mustJob(t, p, "plan-platform-stage-eu-central-1-vpc")
	planSecrets := planJob.Secrets()
	if planSecrets == nil {
		t.Fatal("expected secrets on plan job")
	}
	secret, exists := planSecrets["AWS_SECRET_KEY"]
	if !exists {
		t.Fatal("expected AWS_SECRET_KEY in secrets")
	}
	if !secret.File {
		t.Error("expected secret.File to be true")
	}
	if secret.VaultPath != "secret/data/aws/key@production" {
		t.Errorf("expected VaultPath shorthand, got %q", secret.VaultPath)
	}

	applyJob := mustJob(t, p, "apply-platform-stage-eu-central-1-vpc")
	applySecrets := applyJob.Secrets()
	if applySecrets == nil {
		t.Fatal("expected secrets on apply job")
	}
	if _, exists := applySecrets["AWS_SECRET_KEY"]; !exists {
		t.Error("expected AWS_SECRET_KEY in apply job secrets")
	}
}

func TestGenerator_DetailedPlanForcedByResourceConsumer(t *testing.T) {
	// Regression: a contributor that reads plan.json (e.g. cost) must force the
	// matching plan job to emit plan.json.
	cfg := createTestConfig()
	cfg.Contributions = []*pipeline.Contribution{testContribution(t, pipeline.ContributedJobOptions{
		Name:     "cost-estimation",
		Commands: []string{"terraci cost"},
		Consumes: []pipeline.ResourceRequest{
			pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
		},
	})}

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	depGraph := citest.DependencyGraph([]*discovery.Module{module}, map[string][]string{
		module.ID(): {},
	})

	gen := newTestGenerator(t, cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, []*discovery.Module{module})
	out, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	p, ok := out.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	planJob := mustJob(t, p, "plan-platform-stage-eu-central-1-vpc")
	artifacts := planJob.Artifacts()
	if artifacts == nil {
		t.Fatal("plan job artifacts missing")
	}
	hasPlanJSON := false
	for _, p := range artifacts.Paths {
		if strings.HasSuffix(p, "plan.json") {
			hasPlanJSON = true
			break
		}
	}
	if !hasPlanJSON {
		t.Errorf("PlanJSON consumer did not emit plan.json; artifacts=%v", artifacts.Paths)
	}
	planName := "plan-platform-stage-eu-central-1-vpc"
	if artifacts.Name != pipeline.PlanArtifactName(planName) {
		t.Errorf("plan artifact name = %q, want %q", artifacts.Name, pipeline.PlanArtifactName(planName))
	}
}

func TestGenerator_Generate_WithArtifacts(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.JobDefaults = &JobDefaults{
		Artifacts: &ArtifactsConfig{
			Paths:    []string{"*.json", "reports/"},
			ExpireIn: "1 week",
			When:     "always",
		},
	}

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}
	depGraph := citest.DependencyGraph(modules, map[string][]string{
		modules[0].ID(): {},
	})

	gen := newTestGenerator(t, cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)
	genPipeline, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	planJob := mustJob(t, p, "plan-platform-stage-eu-central-1-vpc")
	artifacts := planJob.Artifacts()
	if artifacts == nil {
		t.Fatal("expected artifacts on plan job")
	}
	if artifacts.ExpireIn != "1 week" {
		t.Errorf("expected ExpireIn '1 week', got %q", artifacts.ExpireIn)
	}
	if artifacts.When != "always" {
		t.Errorf("expected When 'always', got %q", artifacts.When)
	}
	foundJSON := false
	for _, path := range artifacts.Paths {
		if path == "*.json" {
			foundJSON = true
		}
	}
	if !foundJSON {
		t.Error("expected *.json in artifact paths")
	}
}

func TestGenerator_Generate_WithPolicyCheck(t *testing.T) {
	cfg := createTestConfig()
	cfg.Contributions = []*pipeline.Contribution{testContribution(t, pipeline.ContributedJobOptions{
		Name:     "policy-check",
		Commands: []string{"terraci policy check --format text"},
		Consumes: []pipeline.ResourceRequest{
			pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
		},
		Produces: []pipeline.ResourceSpec{
			pipeline.PluginResource(pipeline.ResourceKindPluginResult, "policy", ".terraci/policy-results.json"),
			pipeline.PluginResource(pipeline.ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
		},
		AllowFailure: false,
	})}

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}
	depGraph := citest.DependencyGraph(modules, map[string][]string{
		modules[0].ID(): {},
	})

	gen := newTestGenerator(t, cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)
	genPipeline, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	policyJob := mustJob(t, p, "policy-check")
	stages := p.Stages()
	if policyJob.Stage() != stages[len(stages)-1] {
		t.Errorf("policy-check job stage: expected last DAG stage, got %s in %v", policyJob.Stage(), stages)
	}
	hasCheck := false
	for _, line := range policyJob.Script() {
		if strings.Contains(line, "terraci policy check") {
			hasCheck = true
		}
	}
	if !hasCheck {
		t.Error("expected 'terraci policy check' in policy job script")
	}
	needs := policyJob.Needs()
	if len(needs) == 0 {
		t.Error("expected policy-check job to have needs")
	}
	for _, need := range needs {
		if need.Artifacts == nil || !*need.Artifacts {
			t.Fatalf("need %#v does not explicitly enable artifacts", need)
		}
	}
	artifacts := policyJob.Artifacts()
	if artifacts == nil {
		t.Fatal("policy-check artifacts missing")
	}
	if artifacts.Name != pipeline.ResultArtifactName("policy-check") {
		t.Fatalf("policy-check artifact name = %q, want %q", artifacts.Name, pipeline.ResultArtifactName("policy-check"))
	}
	if !slices.Contains(artifacts.Paths, ".terraci/policy-report.json") {
		t.Fatalf("policy-check artifact paths = %v, want policy report", artifacts.Paths)
	}
}
