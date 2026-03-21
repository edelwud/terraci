package gitlab

import (
	"slices"
	"strings"
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/parser"
	"github.com/edelwud/terraci/internal/pipeline"
	"github.com/edelwud/terraci/pkg/config"
)

// createTestConfig creates a test configuration with default values
func createTestConfig() *config.Config {
	return &config.Config{
		GitLab: &config.GitLabConfig{
			Image: config.Image{
				Name: "hashicorp/terraform:1.6",
			},
			PlanEnabled: true,
		},
	}
}

// createTestDeps creates test dependencies map
func createTestDeps(modules []*discovery.Module, deps map[string][]string) map[string]*parser.ModuleDependencies {
	result := make(map[string]*parser.ModuleDependencies)
	for _, m := range modules {
		modDeps := &parser.ModuleDependencies{
			Module:    m,
			DependsOn: deps[m.ID()],
		}
		result[m.ID()] = modDeps
	}
	return result
}

func TestNewGenerator(t *testing.T) {
	cfg := createTestConfig()
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}
	depGraph := graph.NewDependencyGraph()

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)

	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if gen.config != cfg.GitLab {
		t.Error("config not set correctly")
	}
	if len(gen.modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(gen.modules))
	}
}

func TestGenerator_Generate_SingleModule(t *testing.T) {
	cfg := createTestConfig()
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	var genPipeline pipeline.GeneratedPipeline
	var err error
	genPipeline, err = gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if genPipeline == nil {
		t.Fatal("pipeline is nil")
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	// Should have 2 stages (plan-0, apply-0)
	if len(p.Stages) != 2 {
		t.Errorf("expected 2 stages, got %d: %v", len(p.Stages), p.Stages)
	}

	// Should have 2 jobs (plan + apply)
	if len(p.Jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(p.Jobs))
	}

	// Check job names
	planJobName := "plan-platform-stage-eu-central-1-vpc"
	applyJobName := "apply-platform-stage-eu-central-1-vpc"

	if _, exists := p.Jobs[planJobName]; !exists {
		t.Errorf("missing plan job: %s", planJobName)
	}
	if _, exists := p.Jobs[applyJobName]; !exists {
		t.Errorf("missing apply job: %s", applyJobName)
	}
}

func TestGenerator_Generate_WithDependencies(t *testing.T) {
	cfg := createTestConfig()
	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	modules := []*discovery.Module{vpc, eks}

	// EKS depends on VPC
	deps := createTestDeps(modules, map[string][]string{
		vpc.ID(): {},
		eks.ID(): {vpc.ID()},
	})

	depGraph := graph.BuildFromDependencies(modules, deps)
	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	// Should have 4 stages (plan-0, apply-0, plan-1, apply-1)
	if len(p.Stages) != 4 {
		t.Errorf("expected 4 stages, got %d: %v", len(p.Stages), p.Stages)
	}

	// Check EKS plan job depends on VPC apply
	eksApplyJob := p.Jobs["apply-platform-stage-eu-central-1-eks"]
	if eksApplyJob == nil {
		t.Fatal("EKS apply job not found")
	}

	// EKS apply should need VPC apply
	hasVPCDep := false
	for _, need := range eksApplyJob.Needs {
		if need.Job == "apply-platform-stage-eu-central-1-vpc" {
			hasVPCDep = true
			break
		}
	}
	if !hasVPCDep {
		t.Error("EKS apply job should depend on VPC apply job")
	}
}

func TestGenerator_Generate_PlanOnly(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.PlanOnly = true
	cfg.GitLab.PlanEnabled = true

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	// Should have only 1 stage (plan-0, no apply)
	if len(p.Stages) != 1 {
		t.Errorf("expected 1 stage for plan-only, got %d: %v", len(p.Stages), p.Stages)
	}

	// Should have only 1 job (plan, no apply)
	if len(p.Jobs) != 1 {
		t.Errorf("expected 1 job for plan-only, got %d", len(p.Jobs))
	}

	// Check no apply jobs
	for name := range p.Jobs {
		if strings.HasPrefix(name, "apply-") {
			t.Errorf("unexpected apply job in plan-only mode: %s", name)
		}
	}
}

func TestGenerator_Generate_PlanOnlyWithDependencies(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.PlanOnly = true
	cfg.GitLab.PlanEnabled = true

	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	modules := []*discovery.Module{vpc, eks}

	deps := createTestDeps(modules, map[string][]string{
		vpc.ID(): {},
		eks.ID(): {vpc.ID()},
	})

	depGraph := graph.BuildFromDependencies(modules, deps)
	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	// In plan-only mode, EKS plan should depend on VPC plan (not apply)
	eksPlanJob := p.Jobs["plan-platform-stage-eu-central-1-eks"]
	if eksPlanJob == nil {
		t.Fatal("EKS plan job not found")
	}

	hasVPCPlanDep := false
	for _, need := range eksPlanJob.Needs {
		if need.Job == "plan-platform-stage-eu-central-1-vpc" {
			hasVPCPlanDep = true
		}
		if strings.HasPrefix(need.Job, "apply-") {
			t.Errorf("plan job should not depend on apply job in plan-only mode: %s", need.Job)
		}
	}
	if !hasVPCPlanDep {
		t.Error("EKS plan job should depend on VPC plan job in plan-only mode")
	}
}

func TestGenerator_Generate_AutoApprove(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.AutoApprove = true

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	applyJob := p.Jobs["apply-platform-stage-eu-central-1-vpc"]
	if applyJob == nil {
		t.Fatal("apply job not found")
	}

	// With auto-approve, When should be empty (not "manual")
	if applyJob.When == "manual" {
		t.Error("apply job should not be manual when auto-approve is enabled")
	}
}

func TestGenerator_Generate_ManualApprove(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.AutoApprove = false

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	applyJob := p.Jobs["apply-platform-stage-eu-central-1-vpc"]
	if applyJob == nil {
		t.Fatal("apply job not found")
	}

	// Without auto-approve, When should be "manual"
	if applyJob.When != "manual" {
		t.Errorf("apply job should be manual, got %q", applyJob.When)
	}
}

func TestGenerator_Generate_CustomStagesPrefix(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.StagesPrefix = "terraform"

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	// Stages should use custom prefix
	for _, stage := range p.Stages {
		if !strings.HasPrefix(stage, "terraform-") {
			t.Errorf("stage should have custom prefix 'terraform-', got %s", stage)
		}
	}
}

func TestGenerator_Generate_TerraformBinary(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.TerraformBinary = "tofu"

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	// Check TERRAFORM_BINARY variable
	if p.Variables["TERRAFORM_BINARY"] != "tofu" {
		t.Errorf("expected TERRAFORM_BINARY=tofu, got %s", p.Variables["TERRAFORM_BINARY"])
	}
}

func TestGenerator_Generate_JobVariables(t *testing.T) {
	cfg := createTestConfig()
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
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

	// Check job variables
	expectedVars := map[string]string{
		"TF_MODULE_PATH": "platform/stage/eu-central-1/vpc",
		"TF_SERVICE":     "platform",
		"TF_ENVIRONMENT": "stage",
		"TF_REGION":      "eu-central-1",
		"TF_MODULE":      "vpc",
	}

	for k, v := range expectedVars {
		if planJob.Variables[k] != v {
			t.Errorf("expected %s=%s, got %s", k, v, planJob.Variables[k])
		}
	}
}

func TestGenerator_Generate_ResourceGroup(t *testing.T) {
	cfg := createTestConfig()
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	// Both plan and apply should have the same resource_group
	planJob := p.Jobs["plan-platform-stage-eu-central-1-vpc"]
	applyJob := p.Jobs["apply-platform-stage-eu-central-1-vpc"]

	expectedResourceGroup := "platform/stage/eu-central-1/vpc"

	if planJob.ResourceGroup != expectedResourceGroup {
		t.Errorf("plan job resource_group: expected %s, got %s", expectedResourceGroup, planJob.ResourceGroup)
	}
	if applyJob.ResourceGroup != expectedResourceGroup {
		t.Errorf("apply job resource_group: expected %s, got %s", expectedResourceGroup, applyJob.ResourceGroup)
	}
}

func TestGenerator_DryRun(t *testing.T) {
	cfg := createTestConfig()
	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	modules := []*discovery.Module{vpc, eks}

	deps := createTestDeps(modules, map[string][]string{
		vpc.ID(): {},
		eks.ID(): {vpc.ID()},
	})

	depGraph := graph.BuildFromDependencies(modules, deps)
	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	result, err := gen.DryRun(modules)
	if err != nil {
		t.Fatalf("DryRun failed: %v", err)
	}

	if result.TotalModules != 2 {
		t.Errorf("expected TotalModules=2, got %d", result.TotalModules)
	}
	if result.AffectedModules != 2 {
		t.Errorf("expected AffectedModules=2, got %d", result.AffectedModules)
	}
	if result.Stages != 4 {
		t.Errorf("expected Stages=4, got %d", result.Stages)
	}
	if len(result.ExecutionOrder) != 2 {
		t.Errorf("expected 2 execution levels, got %d", len(result.ExecutionOrder))
	}
}

func TestGenerator_jobName(t *testing.T) {
	cfg := createTestConfig()
	gen := NewGenerator(cfg.GitLab, cfg.Policy, graph.NewDependencyGraph(), nil)

	tests := []struct {
		module   *discovery.Module
		jobType  string
		expected string
	}{
		{
			module:   discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
			jobType:  "plan",
			expected: "plan-platform-stage-eu-central-1-vpc",
		},
		{
			module:   discovery.TestModule("platform", "prod", "us-west-2", "eks"),
			jobType:  "apply",
			expected: "apply-platform-prod-us-west-2-eks",
		},
	}

	for _, tt := range tests {
		result := gen.jobName(tt.module, tt.jobType)
		if result != tt.expected {
			t.Errorf("jobName(%s, %s) = %s, expected %s", tt.module.ID(), tt.jobType, result, tt.expected)
		}
	}
}

func TestPipeline_ToYAML(t *testing.T) {
	p := &Pipeline{
		Stages:    []string{"plan-0", "apply-0"},
		Variables: map[string]string{"TERRAFORM_BINARY": "terraform"},
		Default: &DefaultConfig{
			Image: &ImageConfig{Name: "hashicorp/terraform:1.6"},
		},
		Jobs: map[string]*Job{
			"plan-test": {
				Stage:  "plan-0",
				Script: []string{"terraform plan"},
			},
		},
	}

	yamlBytes, err := p.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	yaml := string(yamlBytes)

	// Check key elements are present
	if !strings.Contains(yaml, "stages:") {
		t.Error("YAML should contain stages")
	}
	if !strings.Contains(yaml, "plan-0") {
		t.Error("YAML should contain plan-0 stage")
	}
	if !strings.Contains(yaml, "TERRAFORM_BINARY") {
		t.Error("YAML should contain TERRAFORM_BINARY variable")
	}
	if !strings.Contains(yaml, "plan-test:") {
		t.Error("YAML should contain plan-test job")
	}
}

func TestGenerator_Generate_WithMRIntegration(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.MR = &config.MRConfig{}

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
		modules[1].ID(): {modules[0].ID()},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	// Check summary job exists
	summaryJob := p.Jobs[SummaryJobName]
	if summaryJob == nil {
		t.Fatal("summary job not found")
	}

	// Check summary job stage
	if summaryJob.Stage != SummaryStageName {
		t.Errorf("summary job stage: expected %s, got %s", SummaryStageName, summaryJob.Stage)
	}

	// Check summary job has correct needs (all plan jobs)
	if len(summaryJob.Needs) != 2 {
		t.Errorf("summary job should have 2 needs, got %d", len(summaryJob.Needs))
	}

	// Check summary stage is in stages list
	if !slices.Contains(p.Stages, SummaryStageName) {
		t.Errorf("stages should contain %s", SummaryStageName)
	}

	// Check plan jobs have plan.txt in artifacts
	planJob := p.Jobs["plan-platform-stage-eu-central-1-vpc"]
	if planJob == nil {
		t.Fatal("plan job not found")
	}
	if planJob.Artifacts == nil {
		t.Fatal("plan job should have artifacts")
	}
	hasPlanTxt := false
	for _, path := range planJob.Artifacts.Paths {
		if strings.Contains(path, "plan.txt") {
			hasPlanTxt = true
			break
		}
	}
	if !hasPlanTxt {
		t.Error("plan job artifacts should include plan.txt")
	}
}

func TestGenerator_Generate_WithMRIntegration_Disabled(t *testing.T) {
	cfg := createTestConfig()
	// No MR config - MR integration disabled

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	// Check summary job does NOT exist
	if _, exists := p.Jobs[SummaryJobName]; exists {
		t.Error("summary job should not exist when MR integration is disabled")
	}

	// Check summary stage is NOT in stages list
	for _, stage := range p.Stages {
		if stage == SummaryStageName {
			t.Error("stages should not contain summary stage when MR is disabled")
		}
	}
}

func TestGenerator_isMREnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected bool
	}{
		{
			name:     "nil MR config",
			config:   &config.Config{GitLab: &config.GitLabConfig{}},
			expected: false,
		},
		{
			name:     "MR config present, no comment config",
			config:   &config.Config{GitLab: &config.GitLabConfig{MR: &config.MRConfig{}}},
			expected: true,
		},
		{
			name:     "MR config present, comment enabled nil",
			config:   &config.Config{GitLab: &config.GitLabConfig{MR: &config.MRConfig{Comment: &config.MRCommentConfig{}}}},
			expected: true,
		},
		{
			name: "MR config present, comment explicitly enabled",
			config: &config.Config{GitLab: &config.GitLabConfig{MR: &config.MRConfig{Comment: &config.MRCommentConfig{
				Enabled: boolPtr(true),
			}}}},
			expected: true,
		},
		{
			name: "MR config present, comment explicitly disabled",
			config: &config.Config{GitLab: &config.GitLabConfig{MR: &config.MRConfig{Comment: &config.MRCommentConfig{
				Enabled: boolPtr(false),
			}}}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewGenerator(tt.config.GitLab, tt.config.Policy, graph.NewDependencyGraph(), nil)
			result := gen.isMREnabled()
			if result != tt.expected {
				t.Errorf("isMREnabled() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestGenerator_Generate_WithSecrets(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitLab.JobDefaults = &config.JobDefaults{
		Secrets: map[string]config.Secret{
			"AWS_SECRET_KEY": {
				Vault: &config.VaultSecret{
					Shorthand: "secret/data/aws/key@production",
				},
				File: true,
			},
		},
	}

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}
	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
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
	cfg.GitLab.JobDefaults = &config.JobDefaults{
		Artifacts: &config.ArtifactsConfig{
			Paths:    []string{"*.json", "reports/"},
			ExpireIn: "1 week",
			When:     "always",
		},
	}

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}
	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
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
	for _, p := range planJob.Artifacts.Paths {
		if p == "*.json" {
			foundJSON = true
		}
	}
	if !foundJSON {
		t.Error("expected *.json in artifact paths")
	}
}

func TestGenerator_Generate_WithPolicyCheck(t *testing.T) {
	cfg := createTestConfig()
	cfg.Policy = &config.PolicyConfig{
		Enabled:   true,
		OnFailure: config.PolicyActionBlock,
		Sources: []config.PolicySource{
			{Path: "policies"},
		},
	}

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
	}
	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitLab, cfg.Policy, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	p, ok := genPipeline.(*Pipeline)
	if !ok {
		t.Fatal("expected *Pipeline type")
	}

	// Check policy-check stage exists
	if !slices.Contains(p.Stages, PolicyCheckStageName) {
		t.Errorf("expected policy-check stage in stages: %v", p.Stages)
	}

	// Check policy-check job exists
	policyJob := p.Jobs[PolicyCheckJobName]
	if policyJob == nil {
		t.Fatal("policy-check job not found")
	}

	if policyJob.Stage != PolicyCheckStageName {
		t.Errorf("expected policy-check job stage=%s, got %s", PolicyCheckStageName, policyJob.Stage)
	}

	// Verify script contains policy commands
	hasCheck := false
	for _, line := range policyJob.Script {
		if strings.Contains(line, "terraci policy check") {
			hasCheck = true
		}
	}
	if !hasCheck {
		t.Error("expected 'terraci policy check' in policy job script")
	}

	// Verify policy job depends on plan jobs
	if len(policyJob.Needs) == 0 {
		t.Error("expected policy-check job to have needs")
	}
}
