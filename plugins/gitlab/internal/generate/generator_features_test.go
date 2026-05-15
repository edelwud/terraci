package generate

import (
	"slices"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestGenerator_Generate_WithSummaryContribution(t *testing.T) {
	cfg := createTestConfig()
	cfg.Contributions = []*pipeline.Contribution{{
		Jobs: []pipeline.ContributedJob{{
			Name:     "terraci-summary",
			Commands: []string{"terraci summary"},
			Consumes: []pipeline.ResourceRequest{
				pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
			},
			AllowFailure: false,
		}},
	}}

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
	if summaryJob.Stage != p.Stages[len(p.Stages)-1] {
		t.Errorf("summary job stage: expected last DAG stage, got %s in %v", summaryJob.Stage, p.Stages)
	}
	if len(summaryJob.Needs) != 2 {
		t.Errorf("summary job should have 2 needs, got %d", len(summaryJob.Needs))
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

	planJob := p.Jobs["plan-platform-stage-eu-central-1-vpc"]
	if planJob == nil {
		t.Fatal("plan job not found")
	}
	if planJob.Secrets == nil {
		t.Fatal("expected secrets on plan job")
	}
	secret, exists := planJob.Secrets["AWS_SECRET_KEY"]
	if !exists {
		t.Fatal("expected AWS_SECRET_KEY in secrets")
	}
	if !secret.File {
		t.Error("expected secret.File to be true")
	}
	if secret.VaultPath != "secret/data/aws/key@production" {
		t.Errorf("expected VaultPath shorthand, got %q", secret.VaultPath)
	}

	applyJob := p.Jobs["apply-platform-stage-eu-central-1-vpc"]
	if applyJob == nil {
		t.Fatal("apply job not found")
	}
	if applyJob.Secrets == nil {
		t.Fatal("expected secrets on apply job")
	}
	if _, exists := applyJob.Secrets["AWS_SECRET_KEY"]; !exists {
		t.Error("expected AWS_SECRET_KEY in apply job secrets")
	}
}

func TestGenerator_DetailedPlanForcedByResourceConsumer(t *testing.T) {
	// Regression: when execution.plan_mode is "standard", a contributor that
	// reads plan.json (e.g. cost) used to be silently broken because the plan
	// job did not emit plan.json. The resource request must lift DetailedPlan.
	cfg := createTestConfig()
	cfg.Contributions = []*pipeline.Contribution{{
		Jobs: []pipeline.ContributedJob{{
			Name:     "cost-estimation",
			Commands: []string{"terraci cost"},
			Consumes: []pipeline.ResourceRequest{
				pipeline.AllPlanResources(pipeline.ResourceKindPlanJSON),
			},
		}},
	}}

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

	planJob := p.Jobs["plan-platform-stage-eu-central-1-vpc"]
	if planJob == nil || planJob.Artifacts == nil {
		t.Fatal("plan job artifacts missing")
	}
	hasPlanJSON := false
	for _, p := range planJob.Artifacts.Paths {
		if strings.HasSuffix(p, "plan.json") {
			hasPlanJSON = true
			break
		}
	}
	if !hasPlanJSON {
		t.Errorf("PlanJSON consumer did not emit plan.json; artifacts=%v", planJob.Artifacts.Paths)
	}
	planName := "plan-platform-stage-eu-central-1-vpc"
	if planJob.Artifacts.Name != pipeline.PlanArtifactName(planName) {
		t.Errorf("plan artifact name = %q, want %q", planJob.Artifacts.Name, pipeline.PlanArtifactName(planName))
	}
}

func TestGenerator_Generate_WithArtifacts(t *testing.T) {
	cfg := createTestConfig()
	cfg.Execution.PlanMode = execution.PlanModeDetailed
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

	planJob := p.Jobs["plan-platform-stage-eu-central-1-vpc"]
	if planJob == nil {
		t.Fatal("plan job not found")
	}
	if planJob.Artifacts == nil {
		t.Fatal("expected artifacts on plan job")
	}
	if planJob.Artifacts.ExpireIn != "1 week" {
		t.Errorf("expected ExpireIn '1 week', got %q", planJob.Artifacts.ExpireIn)
	}
	if planJob.Artifacts.When != "always" {
		t.Errorf("expected When 'always', got %q", planJob.Artifacts.When)
	}
	foundJSON := false
	for _, path := range planJob.Artifacts.Paths {
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
	cfg.Contributions = []*pipeline.Contribution{{
		Jobs: []pipeline.ContributedJob{{
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
		}},
	}}

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

	policyJob := p.Jobs["policy-check"]
	if policyJob == nil {
		t.Fatal("policy-check job not found")
	}
	if policyJob.Stage != p.Stages[len(p.Stages)-1] {
		t.Errorf("policy-check job stage: expected last DAG stage, got %s in %v", policyJob.Stage, p.Stages)
	}
	hasCheck := false
	for _, line := range policyJob.Script {
		if strings.Contains(line, "terraci policy check") {
			hasCheck = true
		}
	}
	if !hasCheck {
		t.Error("expected 'terraci policy check' in policy job script")
	}
	if len(policyJob.Needs) == 0 {
		t.Error("expected policy-check job to have needs")
	}
	for _, need := range policyJob.Needs {
		if need.Artifacts == nil || !*need.Artifacts {
			t.Fatalf("need %#v does not explicitly enable artifacts", need)
		}
	}
	if policyJob.Artifacts == nil {
		t.Fatal("policy-check artifacts missing")
	}
	if policyJob.Artifacts.Name != pipeline.ResultArtifactName("policy-check") {
		t.Fatalf("policy-check artifact name = %q, want %q", policyJob.Artifacts.Name, pipeline.ResultArtifactName("policy-check"))
	}
	if !slices.Contains(policyJob.Artifacts.Paths, ".terraci/policy-report.json") {
		t.Fatalf("policy-check artifact paths = %v, want policy report", policyJob.Artifacts.Paths)
	}
}
