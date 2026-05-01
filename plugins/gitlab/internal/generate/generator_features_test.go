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

func TestGenerator_Generate_WithMRIntegration(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.MR = &MRConfig{}
	cfg.Contributions = []*pipeline.Contribution{{
		Jobs: []pipeline.ContributedJob{{
			Name:          "terraci-summary",
			Phase:         pipeline.PhaseFinalize,
			Commands:      []string{"terraci summary"},
			DependsOnPlan: true,
			AllowFailure:  false,
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

	gen := NewGenerator(cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	assertPipeline(t, p).
		hasJob("terraci-summary").
		hasStage("finalize")
	summaryJob := mustJob(t, p, "terraci-summary")
	if summaryJob.Stage != "finalize" {
		t.Errorf("summary job stage: expected finalize, got %s", summaryJob.Stage)
	}
	if len(summaryJob.Needs) != 2 {
		t.Errorf("summary job should have 2 needs, got %d", len(summaryJob.Needs))
	}
	assertPipeline(t, p).
		job("plan-platform-stage-eu-central-1-vpc").
		artifactPathContains("plan.txt")
}

func TestGenerator_Generate_WithMRIntegration_Disabled(t *testing.T) {
	cfg := createTestConfig()
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	depGraph := citest.DependencyGraph(modules, map[string][]string{
		modules[0].ID(): {},
	})

	gen := NewGenerator(cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	assertPipeline(t, p).
		noJob("terraci-summary").
		noStage("finalize")
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

	gen := NewGenerator(cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
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

	gen := NewGenerator(cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
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
			Name:          "policy-check",
			Phase:         pipeline.PhasePostPlan,
			Commands:      []string{"terraci policy pull", "terraci policy check"},
			ArtifactPaths: []string{".terraci/policy-results.json"},
			DependsOnPlan: true,
			AllowFailure:  false,
		}},
	}}

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}
	depGraph := citest.DependencyGraph(modules, map[string][]string{
		modules[0].ID(): {},
	})

	gen := NewGenerator(cfg.GitLab, cfg.Execution, cfg.Contributions, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	if !slices.Contains(p.Stages, "post-plan") {
		t.Errorf("expected post-plan stage in stages: %v", p.Stages)
	}

	policyJob := p.Jobs["policy-check"]
	if policyJob == nil {
		t.Fatal("policy-check job not found")
	}
	if policyJob.Stage != "post-plan" {
		t.Errorf("expected policy-check job stage=post-plan, got %s", policyJob.Stage)
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
}
