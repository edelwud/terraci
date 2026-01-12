// Package e2e provides end-to-end tests for terraci pipeline generation
package e2e

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/graph"
	"github.com/edelwud/terraci/internal/parser"
	"github.com/edelwud/terraci/internal/pipeline/gitlab"
	"github.com/edelwud/terraci/pkg/config"
)

// createTestModules creates a standard test module set with dependencies:
// Level 0: vpc, s3
// Level 1: eks (depends on vpc), rds (depends on vpc)
// Level 2: app (depends on eks, rds, s3)
func createTestModules() []*discovery.Module {
	return []*discovery.Module{
		{Service: "platform", Environment: "stage", Region: "eu-central-1", Module: "vpc", RelativePath: "platform/stage/eu-central-1/vpc"},
		{Service: "platform", Environment: "stage", Region: "eu-central-1", Module: "s3", RelativePath: "platform/stage/eu-central-1/s3"},
		{Service: "platform", Environment: "stage", Region: "eu-central-1", Module: "eks", RelativePath: "platform/stage/eu-central-1/eks"},
		{Service: "platform", Environment: "stage", Region: "eu-central-1", Module: "rds", RelativePath: "platform/stage/eu-central-1/rds"},
		{Service: "platform", Environment: "stage", Region: "eu-central-1", Module: "app", RelativePath: "platform/stage/eu-central-1/app"},
	}
}

func createTestDependencies() map[string]*parser.ModuleDependencies {
	return map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-central-1/s3":  {DependsOn: []string{}},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/rds": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/app": {DependsOn: []string{
			"platform/stage/eu-central-1/eks",
			"platform/stage/eu-central-1/rds",
			"platform/stage/eu-central-1/s3",
		}},
	}
}

func TestPipelineGeneration_Basic(t *testing.T) {
	modules := createTestModules()
	deps := createTestDependencies()
	depGraph := graph.BuildFromDependencies(modules, deps)

	cfg := config.DefaultConfig()
	cfg.GitLab.PlanEnabled = true
	cfg.GitLab.AutoApprove = false

	generator := gitlab.NewGenerator(cfg, depGraph, modules)
	result, err := generator.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have both plan and apply stages
	hasplanStage := false
	hasApplyStage := false
	for _, stage := range pipeline.Stages {
		if strings.HasPrefix(stage, "deploy-plan-") {
			hasplanStage = true
		}
		if strings.HasPrefix(stage, "deploy-apply-") {
			hasApplyStage = true
		}
	}

	if !hasplanStage {
		t.Error("Expected plan stages in pipeline")
	}
	if !hasApplyStage {
		t.Error("Expected apply stages in pipeline")
	}

	// Should have both plan and apply jobs for each module
	expectedPlanJobs := []string{
		"plan-platform-stage-eu-central-1-vpc",
		"plan-platform-stage-eu-central-1-s3",
		"plan-platform-stage-eu-central-1-eks",
		"plan-platform-stage-eu-central-1-rds",
		"plan-platform-stage-eu-central-1-app",
	}

	expectedApplyJobs := []string{
		"apply-platform-stage-eu-central-1-vpc",
		"apply-platform-stage-eu-central-1-s3",
		"apply-platform-stage-eu-central-1-eks",
		"apply-platform-stage-eu-central-1-rds",
		"apply-platform-stage-eu-central-1-app",
	}

	for _, jobName := range expectedPlanJobs {
		if _, exists := pipeline.Jobs[jobName]; !exists {
			t.Errorf("Expected plan job %s not found", jobName)
		}
	}

	for _, jobName := range expectedApplyJobs {
		if _, exists := pipeline.Jobs[jobName]; !exists {
			t.Errorf("Expected apply job %s not found", jobName)
		}
	}
}

func TestPipelineGeneration_PlanOnly(t *testing.T) {
	modules := createTestModules()
	deps := createTestDependencies()
	depGraph := graph.BuildFromDependencies(modules, deps)

	cfg := config.DefaultConfig()
	cfg.GitLab.PlanEnabled = true
	cfg.GitLab.PlanOnly = true

	generator := gitlab.NewGenerator(cfg, depGraph, modules)
	result, err := generator.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have only plan stages, no apply stages
	for _, stage := range pipeline.Stages {
		if strings.HasPrefix(stage, "deploy-apply-") {
			t.Errorf("Unexpected apply stage in plan-only mode: %s", stage)
		}
	}

	// Should have only plan jobs, no apply jobs
	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "apply-") {
			t.Errorf("Unexpected apply job in plan-only mode: %s", jobName)
		}
	}

	// Plan jobs should exist
	expectedPlanJobs := []string{
		"plan-platform-stage-eu-central-1-vpc",
		"plan-platform-stage-eu-central-1-s3",
		"plan-platform-stage-eu-central-1-eks",
		"plan-platform-stage-eu-central-1-rds",
		"plan-platform-stage-eu-central-1-app",
	}

	for _, jobName := range expectedPlanJobs {
		if _, exists := pipeline.Jobs[jobName]; !exists {
			t.Errorf("Expected plan job %s not found", jobName)
		}
	}
}

func TestPipelineGeneration_PlanOnlyNeeds(t *testing.T) {
	modules := createTestModules()
	deps := createTestDependencies()
	depGraph := graph.BuildFromDependencies(modules, deps)

	cfg := config.DefaultConfig()
	cfg.GitLab.PlanEnabled = true
	cfg.GitLab.PlanOnly = true

	generator := gitlab.NewGenerator(cfg, depGraph, modules)
	result, err := generator.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// In plan-only mode, plan jobs should depend on plan jobs (not apply jobs)
	eksJob := pipeline.Jobs["plan-platform-stage-eu-central-1-eks"]
	if eksJob == nil {
		t.Fatal("EKS plan job not found")
	}

	// EKS depends on VPC, so should have need for plan-vpc
	foundVPCPlanNeed := false
	for _, need := range eksJob.Needs {
		if need.Job == "plan-platform-stage-eu-central-1-vpc" {
			foundVPCPlanNeed = true
		}
		if strings.HasPrefix(need.Job, "apply-") {
			t.Errorf("Plan job should not depend on apply job in plan-only mode: %s", need.Job)
		}
	}

	if !foundVPCPlanNeed {
		t.Error("EKS plan job should depend on VPC plan job in plan-only mode")
	}

	// App depends on eks, rds, s3 - check all are plan jobs
	appJob := pipeline.Jobs["plan-platform-stage-eu-central-1-app"]
	if appJob == nil {
		t.Fatal("App plan job not found")
	}

	expectedNeeds := map[string]bool{
		"plan-platform-stage-eu-central-1-eks": false,
		"plan-platform-stage-eu-central-1-rds": false,
		"plan-platform-stage-eu-central-1-s3":  false,
	}

	for _, need := range appJob.Needs {
		if _, expected := expectedNeeds[need.Job]; expected {
			expectedNeeds[need.Job] = true
		}
		if strings.HasPrefix(need.Job, "apply-") {
			t.Errorf("Plan job should not depend on apply job: %s", need.Job)
		}
	}

	for needJob, found := range expectedNeeds {
		if !found {
			t.Errorf("App plan job should depend on %s", needJob)
		}
	}
}

func TestPipelineGeneration_ChangedOnlyFilteredNeeds(t *testing.T) {
	modules := createTestModules()
	deps := createTestDependencies()
	depGraph := graph.BuildFromDependencies(modules, deps)

	cfg := config.DefaultConfig()
	cfg.GitLab.PlanEnabled = true

	// Simulate --changed-only scenario: only eks and app changed
	// eks depends on vpc (not changed), app depends on eks, rds, s3 (not changed)
	changedModules := []*discovery.Module{
		modules[2], // eks
		modules[4], // app
	}

	generator := gitlab.NewGenerator(cfg, depGraph, modules)
	result, err := generator.Generate(changedModules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should only have jobs for changed modules
	if len(pipeline.Jobs) != 4 { // 2 plan + 2 apply
		t.Errorf("Expected 4 jobs (2 plan + 2 apply), got %d", len(pipeline.Jobs))
	}

	// VPC, S3, RDS jobs should NOT exist
	unexpectedJobs := []string{
		"plan-platform-stage-eu-central-1-vpc",
		"apply-platform-stage-eu-central-1-vpc",
		"plan-platform-stage-eu-central-1-s3",
		"apply-platform-stage-eu-central-1-s3",
		"plan-platform-stage-eu-central-1-rds",
		"apply-platform-stage-eu-central-1-rds",
	}

	for _, jobName := range unexpectedJobs {
		if _, exists := pipeline.Jobs[jobName]; exists {
			t.Errorf("Unexpected job for non-changed module: %s", jobName)
		}
	}

	// EKS and App jobs should exist
	expectedJobs := []string{
		"plan-platform-stage-eu-central-1-eks",
		"apply-platform-stage-eu-central-1-eks",
		"plan-platform-stage-eu-central-1-app",
		"apply-platform-stage-eu-central-1-app",
	}

	for _, jobName := range expectedJobs {
		if _, exists := pipeline.Jobs[jobName]; !exists {
			t.Errorf("Expected job for changed module not found: %s", jobName)
		}
	}

	// EKS plan job should NOT have vpc in needs (vpc not in target modules)
	eksPlanJob := pipeline.Jobs["plan-platform-stage-eu-central-1-eks"]
	for _, need := range eksPlanJob.Needs {
		if strings.Contains(need.Job, "vpc") {
			t.Errorf("EKS job should not reference vpc job when vpc is not in target modules: %s", need.Job)
		}
	}

	// App plan job should NOT have s3 or rds in needs
	appPlanJob := pipeline.Jobs["plan-platform-stage-eu-central-1-app"]
	for _, need := range appPlanJob.Needs {
		if strings.Contains(need.Job, "s3") {
			t.Errorf("App job should not reference s3 job when s3 is not in target modules: %s", need.Job)
		}
		if strings.Contains(need.Job, "rds") {
			t.Errorf("App job should not reference rds job when rds is not in target modules: %s", need.Job)
		}
	}

	// App should still depend on eks (which IS in target modules)
	foundEksNeed := false
	for _, need := range appPlanJob.Needs {
		if strings.Contains(need.Job, "eks") {
			foundEksNeed = true
		}
	}
	if !foundEksNeed {
		t.Error("App plan job should depend on eks apply job")
	}
}

func TestPipelineGeneration_ChangedOnlyPlanOnly(t *testing.T) {
	modules := createTestModules()
	deps := createTestDependencies()
	depGraph := graph.BuildFromDependencies(modules, deps)

	cfg := config.DefaultConfig()
	cfg.GitLab.PlanEnabled = true
	cfg.GitLab.PlanOnly = true

	// Only eks and app changed
	changedModules := []*discovery.Module{
		modules[2], // eks
		modules[4], // app
	}

	generator := gitlab.NewGenerator(cfg, depGraph, modules)
	result, err := generator.Generate(changedModules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should only have plan jobs (no apply)
	if len(pipeline.Jobs) != 2 {
		t.Errorf("Expected 2 plan jobs, got %d", len(pipeline.Jobs))
	}

	// No apply jobs
	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "apply-") {
			t.Errorf("Unexpected apply job in plan-only mode: %s", jobName)
		}
	}

	// App plan job should depend on eks plan job (not apply)
	appJob := pipeline.Jobs["plan-platform-stage-eu-central-1-app"]
	if appJob == nil {
		t.Fatal("App plan job not found")
	}

	foundEksPlanNeed := false
	for _, need := range appJob.Needs {
		if need.Job == "plan-platform-stage-eu-central-1-eks" {
			foundEksPlanNeed = true
		}
		if strings.HasPrefix(need.Job, "apply-") {
			t.Errorf("Plan job should not depend on apply job: %s", need.Job)
		}
	}

	if !foundEksPlanNeed {
		t.Error("App plan job should depend on eks plan job")
	}
}

func TestPipelineGeneration_ApplyDependsOnPlan(t *testing.T) {
	modules := createTestModules()
	deps := createTestDependencies()
	depGraph := graph.BuildFromDependencies(modules, deps)

	cfg := config.DefaultConfig()
	cfg.GitLab.PlanEnabled = true

	generator := gitlab.NewGenerator(cfg, depGraph, modules)
	result, err := generator.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Each apply job should depend on its own plan job
	for _, module := range modules {
		moduleID := strings.ReplaceAll(module.ID(), "/", "-")
		applyJobName := "apply-" + moduleID
		planJobName := "plan-" + moduleID

		applyJob := pipeline.Jobs[applyJobName]
		if applyJob == nil {
			t.Fatalf("Apply job %s not found", applyJobName)
		}

		foundOwnPlan := false
		for _, need := range applyJob.Needs {
			if need.Job == planJobName {
				foundOwnPlan = true
				break
			}
		}

		if !foundOwnPlan {
			t.Errorf("Apply job %s should depend on its plan job %s", applyJobName, planJobName)
		}
	}
}

func TestPipelineGeneration_NoPlanEnabled(t *testing.T) {
	modules := createTestModules()
	deps := createTestDependencies()
	depGraph := graph.BuildFromDependencies(modules, deps)

	cfg := config.DefaultConfig()
	cfg.GitLab.PlanEnabled = false

	generator := gitlab.NewGenerator(cfg, depGraph, modules)
	result, err := generator.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have no plan jobs
	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "plan-") {
			t.Errorf("Unexpected plan job when PlanEnabled=false: %s", jobName)
		}
	}

	// Should have no plan stages
	for _, stage := range pipeline.Stages {
		if strings.Contains(stage, "-plan-") {
			t.Errorf("Unexpected plan stage when PlanEnabled=false: %s", stage)
		}
	}

	// Apply jobs should NOT depend on plan jobs
	for jobName, job := range pipeline.Jobs {
		for _, need := range job.Needs {
			if strings.HasPrefix(need.Job, "plan-") {
				t.Errorf("Apply job %s should not depend on plan job when PlanEnabled=false: %s", jobName, need.Job)
			}
		}
	}
}

func TestPipelineGeneration_DependencyOrder(t *testing.T) {
	modules := createTestModules()
	deps := createTestDependencies()
	depGraph := graph.BuildFromDependencies(modules, deps)

	cfg := config.DefaultConfig()
	cfg.GitLab.PlanEnabled = true

	generator := gitlab.NewGenerator(cfg, depGraph, modules)
	result, err := generator.Generate(modules)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Extract stage indices
	stageIndex := make(map[string]int)
	for i, stage := range pipeline.Stages {
		stageIndex[stage] = i
	}

	// Verify dependency order: dependencies should be in earlier stages
	// vpc (level 0) -> eks (level 1) -> app (level 2)
	vpcApplyJob := pipeline.Jobs["apply-platform-stage-eu-central-1-vpc"]
	eksApplyJob := pipeline.Jobs["apply-platform-stage-eu-central-1-eks"]
	appApplyJob := pipeline.Jobs["apply-platform-stage-eu-central-1-app"]

	if vpcApplyJob == nil || eksApplyJob == nil || appApplyJob == nil {
		t.Fatal("Expected jobs not found")
	}

	vpcStageIdx := stageIndex[vpcApplyJob.Stage]
	eksStageIdx := stageIndex[eksApplyJob.Stage]
	appStageIdx := stageIndex[appApplyJob.Stage]

	if vpcStageIdx >= eksStageIdx {
		t.Errorf("VPC should be in earlier stage than EKS: vpc=%d, eks=%d", vpcStageIdx, eksStageIdx)
	}

	if eksStageIdx >= appStageIdx {
		t.Errorf("EKS should be in earlier stage than App: eks=%d, app=%d", eksStageIdx, appStageIdx)
	}
}

func TestPipelineGeneration_EmptyModules(t *testing.T) {
	modules := createTestModules()
	deps := createTestDependencies()
	depGraph := graph.BuildFromDependencies(modules, deps)

	cfg := config.DefaultConfig()

	generator := gitlab.NewGenerator(cfg, depGraph, modules)

	// Generate with empty target modules should use all modules
	result, err := generator.Generate(nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have jobs for all modules
	if len(pipeline.Jobs) != 10 { // 5 plan + 5 apply
		t.Errorf("Expected 10 jobs, got %d", len(pipeline.Jobs))
	}
}

func TestPipelineGeneration_SingleModule(t *testing.T) {
	modules := createTestModules()
	deps := createTestDependencies()
	depGraph := graph.BuildFromDependencies(modules, deps)

	cfg := config.DefaultConfig()
	cfg.GitLab.PlanEnabled = true

	generator := gitlab.NewGenerator(cfg, depGraph, modules)

	// Generate for single module (vpc - no dependencies)
	singleModule := []*discovery.Module{modules[0]} // vpc
	result, err := generator.Generate(singleModule)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have only 2 jobs (plan + apply for vpc)
	if len(pipeline.Jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(pipeline.Jobs))
	}

	// VPC jobs should have no needs (no dependencies)
	vpcPlanJob := pipeline.Jobs["plan-platform-stage-eu-central-1-vpc"]
	if vpcPlanJob == nil {
		t.Fatal("VPC plan job not found")
	}

	if len(vpcPlanJob.Needs) != 0 {
		t.Errorf("VPC plan job should have no needs, got %d", len(vpcPlanJob.Needs))
	}
}
