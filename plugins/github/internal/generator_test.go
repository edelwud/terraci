package githubci

import (
	"slices"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

// createTestModule creates a test module with the given parameters
func createTestModule(service, env, region, module string) *discovery.Module {
	return discovery.TestModule(service, env, region, module)
}

// testCfg is a local wrapper used by tests to hold both github and contributed pipeline data.
type testCfg struct {
	GitHub *Config
	Steps  []plugin.PipelineStep
	Jobs   []plugin.PipelineJob
}

// createTestConfig creates a test configuration with default values
func createTestConfig() *testCfg {
	return &testCfg{
		GitHub: &Config{
			RunsOn:      "ubuntu-latest",
			PlanEnabled: true,
			InitEnabled: true,
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

func TestGenerate_SingleModule(t *testing.T) {
	cfg := createTestConfig()
	modules := []*discovery.Module{
		createTestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	var genPipeline pipeline.GeneratedPipeline
	var err error
	genPipeline, err = gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if genPipeline == nil {
		t.Fatal("pipeline is nil")
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	// Should have 2 jobs (plan + apply)
	if len(w.Jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(w.Jobs))
	}

	planJobName := "plan-platform-stage-eu-central-1-vpc"
	applyJobName := "apply-platform-stage-eu-central-1-vpc"

	planJob, exists := w.Jobs[planJobName]
	if !exists {
		t.Fatalf("missing plan job: %s", planJobName)
	}
	if _, exists := w.Jobs[applyJobName]; !exists {
		t.Errorf("missing apply job: %s", applyJobName)
	}

	// Plan job should have Checkout step
	if len(planJob.Steps) == 0 {
		t.Fatal("plan job has no steps")
	}
	if planJob.Steps[0].Name != "Checkout" {
		t.Errorf("first step should be Checkout, got %q", planJob.Steps[0].Name)
	}
	if planJob.Steps[0].Uses != "actions/checkout@v4" {
		t.Errorf("checkout step should use actions/checkout@v4, got %q", planJob.Steps[0].Uses)
	}

	// Plan job should have a step with init and plan commands
	hasInitCmd := false
	hasPlanCmd := false
	hasUploadArtifact := false
	for _, step := range planJob.Steps {
		if strings.Contains(step.Run, "${TERRAFORM_BINARY} init") {
			hasInitCmd = true
		}
		if strings.Contains(step.Run, "${TERRAFORM_BINARY} plan") {
			hasPlanCmd = true
		}
		if strings.Contains(step.Uses, "actions/upload-artifact") {
			hasUploadArtifact = true
		}
	}
	if !hasInitCmd {
		t.Error("plan job should have terraform init command")
	}
	if !hasPlanCmd {
		t.Error("plan job should have terraform plan command")
	}
	if !hasUploadArtifact {
		t.Error("plan job should have upload artifact step")
	}
}

func TestGenerate_WithDependencies(t *testing.T) {
	cfg := createTestConfig()
	vpc := createTestModule("platform", "stage", "eu-central-1", "vpc")
	eks := createTestModule("platform", "stage", "eu-central-1", "eks")
	modules := []*discovery.Module{vpc, eks}

	// EKS depends on VPC
	deps := createTestDeps(modules, map[string][]string{
		vpc.ID(): {},
		eks.ID(): {vpc.ID()},
	})

	depGraph := graph.BuildFromDependencies(modules, deps)
	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	// EKS apply should need VPC apply
	eksApplyJob := w.Jobs["apply-platform-stage-eu-central-1-eks"]
	if eksApplyJob == nil {
		t.Fatal("EKS apply job not found")
	}

	if !slices.Contains(eksApplyJob.Needs, "apply-platform-stage-eu-central-1-vpc") {
		t.Errorf("EKS apply job should depend on VPC apply job, needs: %v", eksApplyJob.Needs)
	}
}

func TestGenerate_PlanOnly(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitHub.PlanOnly = true
	cfg.GitHub.PlanEnabled = true

	modules := []*discovery.Module{
		createTestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	// Should have only 1 job (plan, no apply)
	if len(w.Jobs) != 1 {
		t.Errorf("expected 1 job for plan-only, got %d", len(w.Jobs))
	}

	// Check no apply jobs
	for name := range w.Jobs {
		if strings.HasPrefix(name, "apply-") {
			t.Errorf("unexpected apply job in plan-only mode: %s", name)
		}
	}
}

func TestGenerate_PlanOnlyWithDeps(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitHub.PlanOnly = true
	cfg.GitHub.PlanEnabled = true

	vpc := createTestModule("platform", "stage", "eu-central-1", "vpc")
	eks := createTestModule("platform", "stage", "eu-central-1", "eks")
	modules := []*discovery.Module{vpc, eks}

	deps := createTestDeps(modules, map[string][]string{
		vpc.ID(): {},
		eks.ID(): {vpc.ID()},
	})

	depGraph := graph.BuildFromDependencies(modules, deps)
	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	// In plan-only mode, EKS plan should depend on VPC plan (not apply)
	eksPlanJob := w.Jobs["plan-platform-stage-eu-central-1-eks"]
	if eksPlanJob == nil {
		t.Fatal("EKS plan job not found")
	}

	hasVPCPlanDep := false
	for _, need := range eksPlanJob.Needs {
		if need == "plan-platform-stage-eu-central-1-vpc" {
			hasVPCPlanDep = true
		}
		if strings.HasPrefix(need, "apply-") {
			t.Errorf("plan job should not depend on apply job in plan-only mode: %s", need)
		}
	}
	if !hasVPCPlanDep {
		t.Error("EKS plan job should depend on VPC plan job in plan-only mode")
	}
}

func TestGenerate_AutoApprove(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitHub.AutoApprove = true

	modules := []*discovery.Module{
		createTestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	applyJob := w.Jobs["apply-platform-stage-eu-central-1-vpc"]
	if applyJob == nil {
		t.Fatal("apply job not found")
	}

	// With auto-approve, Environment should be empty
	if applyJob.Environment != "" {
		t.Errorf("apply job should have no environment when auto-approve is enabled, got %q", applyJob.Environment)
	}
}

func TestGenerate_ManualApprove(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitHub.AutoApprove = false

	modules := []*discovery.Module{
		createTestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	applyJob := w.Jobs["apply-platform-stage-eu-central-1-vpc"]
	if applyJob == nil {
		t.Fatal("apply job not found")
	}

	// Without auto-approve, Environment should be "production"
	if applyJob.Environment != "production" {
		t.Errorf("apply job should have environment 'production', got %q", applyJob.Environment)
	}
}

func TestGenerate_CustomBinary(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitHub.TerraformBinary = "tofu"

	modules := []*discovery.Module{
		createTestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	// Workflow env should have TERRAFORM_BINARY=tofu
	if w.Env["TERRAFORM_BINARY"] != "tofu" {
		t.Errorf("expected TERRAFORM_BINARY=tofu, got %s", w.Env["TERRAFORM_BINARY"])
	}
}

func TestGenerate_WithPR(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitHub.PR = &PRConfig{}

	modules := []*discovery.Module{
		createTestModule("platform", "stage", "eu-central-1", "vpc"),
		createTestModule("platform", "stage", "eu-central-1", "eks"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
		modules[1].ID(): {modules[0].ID()},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	// Summary job should exist
	summaryJob := w.Jobs[SummaryJobName]
	if summaryJob == nil {
		t.Fatal("summary job not found")
	}

	// Summary job should need all plan jobs
	if len(summaryJob.Needs) != 2 {
		t.Errorf("summary job should have 2 needs, got %d: %v", len(summaryJob.Needs), summaryJob.Needs)
	}

	hasPlanVPC := false
	hasPlanEKS := false
	for _, need := range summaryJob.Needs {
		if need == "plan-platform-stage-eu-central-1-vpc" {
			hasPlanVPC = true
		}
		if need == "plan-platform-stage-eu-central-1-eks" {
			hasPlanEKS = true
		}
	}
	if !hasPlanVPC {
		t.Error("summary job should need VPC plan job")
	}
	if !hasPlanEKS {
		t.Error("summary job should need EKS plan job")
	}
}

func TestGenerate_WithPolicy(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitHub.PlanEnabled = true
	cfg.Jobs = []plugin.PipelineJob{{
		Name:          "policy-check",
		Stage:         "post-plan",
		Commands:      []string{"terraci policy pull", "terraci policy check"},
		ArtifactPaths: []string{".terraci/policy-results.json"},
		DependsOnPlan: true,
		AllowFailure:  false,
	}}

	modules := []*discovery.Module{
		createTestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	// Policy check job should exist
	policyJob := w.Jobs["policy-check"]
	if policyJob == nil {
		t.Fatal("policy-check job not found")
	}

	// Policy job should need the plan job
	hasPlanDep := false
	for _, need := range policyJob.Needs {
		if need == "plan-platform-stage-eu-central-1-vpc" {
			hasPlanDep = true
		}
	}
	if !hasPlanDep {
		t.Error("policy-check job should depend on plan job")
	}
}

func TestGenerate_WithContainer(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitHub.Container = &Image{
		Name: "hashicorp/terraform:1.6",
	}

	modules := []*discovery.Module{
		createTestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	planJob := w.Jobs["plan-platform-stage-eu-central-1-vpc"]
	if planJob == nil {
		t.Fatal("plan job not found")
	}

	if planJob.Container == nil {
		t.Fatal("plan job should have container set")
	}
	if planJob.Container.Image != "hashicorp/terraform:1.6" {
		t.Errorf("expected container image hashicorp/terraform:1.6, got %s", planJob.Container.Image)
	}
}

func TestGenerate_StepsBefore(t *testing.T) {
	cfg := createTestConfig()
	cfg.GitHub.JobDefaults = &JobDefaults{
		StepsBefore: []ConfigStep{
			{Name: "Setup AWS credentials", Uses: "aws-actions/configure-aws-credentials@v4"},
		},
	}

	modules := []*discovery.Module{
		createTestModule("platform", "stage", "eu-central-1", "vpc"),
	}

	deps := createTestDeps(modules, map[string][]string{
		modules[0].ID(): {},
	})
	depGraph := graph.BuildFromDependencies(modules, deps)

	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
	genPipeline, err := gen.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	w, ok := genPipeline.(*Workflow)
	if !ok {
		t.Fatal("expected *Workflow type")
	}

	planJob := w.Jobs["plan-platform-stage-eu-central-1-vpc"]
	if planJob == nil {
		t.Fatal("plan job not found")
	}

	// Steps should be: Checkout, StepsBefore (Setup AWS), Plan step, Upload artifact
	// Find the setup step and verify it appears before the plan step
	setupIdx := -1
	planIdx := -1
	for i, step := range planJob.Steps {
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
	cfg := createTestConfig()
	vpc := createTestModule("platform", "stage", "eu-central-1", "vpc")
	eks := createTestModule("platform", "stage", "eu-central-1", "eks")
	modules := []*discovery.Module{vpc, eks}

	deps := createTestDeps(modules, map[string][]string{
		vpc.ID(): {},
		eks.ID(): {vpc.ID()},
	})

	depGraph := graph.BuildFromDependencies(modules, deps)
	gen := NewGenerator(cfg.GitHub, cfg.Steps, cfg.Jobs, depGraph, modules)
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
	// 2 modules with PlanEnabled=true -> 4 jobs (plan+apply per module)
	if result.Jobs != 4 {
		t.Errorf("expected Jobs=4, got %d", result.Jobs)
	}
	// 2 execution levels
	if result.Stages != 2 {
		t.Errorf("expected Stages=2, got %d", result.Stages)
	}
	if len(result.ExecutionOrder) != 2 {
		t.Errorf("expected 2 execution levels, got %d", len(result.ExecutionOrder))
	}
}

func TestJobName(t *testing.T) {
	tests := []struct {
		name     string
		module   *discovery.Module
		jobType  string
		expected string
	}{
		{
			name:     "plan job for vpc",
			module:   createTestModule("platform", "stage", "eu-central-1", "vpc"),
			jobType:  "plan",
			expected: "plan-platform-stage-eu-central-1-vpc",
		},
		{
			name:     "apply job for eks",
			module:   createTestModule("platform", "prod", "us-west-2", "eks"),
			jobType:  "apply",
			expected: "apply-platform-prod-us-west-2-eks",
		},
		{
			name:     "plan job with different service",
			module:   createTestModule("data", "dev", "ap-southeast-1", "rds"),
			jobType:  "plan",
			expected: "plan-data-dev-ap-southeast-1-rds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pipeline.JobName(tt.jobType, tt.module)
			if result != tt.expected {
				t.Errorf("jobName(%s, %s) = %s, expected %s", tt.module.ID(), tt.jobType, result, tt.expected)
			}
		})
	}
}

func TestToYAML(t *testing.T) {
	w := &Workflow{
		Name: "Terraform",
		On: WorkflowTrigger{
			Push: &PushTrigger{Branches: []string{"main"}},
		},
		Env: map[string]string{"TERRAFORM_BINARY": "terraform"},
		Jobs: map[string]*Job{
			"plan-test": {
				RunsOn: "ubuntu-latest",
				Steps: []Step{
					{Name: "Checkout", Uses: "actions/checkout@v4"},
					{Name: "Plan", Run: "terraform plan"},
				},
			},
		},
	}

	yamlBytes, err := w.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML failed: %v", err)
	}

	yaml := string(yamlBytes)

	checks := []struct {
		desc     string
		contains string
	}{
		{"generated header", "Generated by terraci"},
		{"name field", "name:"},
		{"jobs field", "jobs:"},
		{"plan-test job", "plan-test:"},
		{"TERRAFORM_BINARY", "TERRAFORM_BINARY"},
	}

	for _, check := range checks {
		if !strings.Contains(yaml, check.contains) {
			t.Errorf("YAML should contain %s (%q)", check.desc, check.contains)
		}
	}
}
