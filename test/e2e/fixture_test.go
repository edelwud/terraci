// Package e2e provides end-to-end tests for terraci pipeline generation
package e2e

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/pipeline/gitlab"
	"github.com/edelwud/terraci/pkg/config"
)

// TestFixture_Basic tests basic pipeline generation using real terraform fixtures
func TestFixture_Basic(t *testing.T) {
	fixture := LoadFixture(t, "basic")

	// Should discover 6 modules (5 stage + 1 prod)
	if len(fixture.Modules) != 6 {
		t.Errorf("expected 6 modules, got %d", len(fixture.Modules))
	}

	// Generate pipeline
	result, err := fixture.Generator.Generate(fixture.Modules)
	if err != nil {
		t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have 12 jobs (6 plan + 6 apply)
	AssertJobCount(t, pipeline, 12)

	// Check specific jobs exist
	AssertJobExists(t, pipeline, "plan-platform-stage-eu-central-1-vpc")
	AssertJobExists(t, pipeline, "apply-platform-stage-eu-central-1-vpc")
	AssertJobExists(t, pipeline, "plan-platform-stage-eu-central-1-eks")
	AssertJobExists(t, pipeline, "apply-platform-stage-eu-central-1-eks")
	AssertJobExists(t, pipeline, "plan-platform-prod-eu-central-1-vpc")
	AssertJobExists(t, pipeline, "apply-platform-prod-eu-central-1-vpc")
}

// TestFixture_BasicDependencies tests that dependencies are correctly resolved
func TestFixture_BasicDependencies(t *testing.T) {
	fixture := LoadFixture(t, "basic")

	result, err := fixture.Generator.Generate(fixture.Modules)
	if err != nil {
		t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// EKS plan should depend on VPC apply
	AssertJobHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-eks", "apply-platform-stage-eu-central-1-vpc")

	// RDS plan should depend on VPC apply
	AssertJobHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-rds", "apply-platform-stage-eu-central-1-vpc")

	// App plan should depend on EKS apply, RDS apply, and S3 apply
	AssertJobHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-app", "apply-platform-stage-eu-central-1-eks")
	AssertJobHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-app", "apply-platform-stage-eu-central-1-rds")
	AssertJobHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-app", "apply-platform-stage-eu-central-1-s3")

	// VPC has no dependencies
	vpcPlanNeeds := GetJobNeeds(pipeline, "plan-platform-stage-eu-central-1-vpc")
	if len(vpcPlanNeeds) != 0 {
		t.Errorf("VPC plan should have no needs, got %v", vpcPlanNeeds)
	}
}

// TestFixture_PlanOnly tests plan-only mode with fixtures
func TestFixture_PlanOnly(t *testing.T) {
	fixture := LoadFixtureWithConfig(t, "basic", func(cfg *config.Config) {
		cfg.GitLab.PlanOnly = true
		cfg.GitLab.PlanEnabled = true
	})

	result, err := fixture.Generator.Generate(fixture.Modules)
	if err != nil {
		t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have only plan jobs (6 modules = 6 plan jobs)
	AssertJobCount(t, pipeline, 6)

	// No apply jobs should exist
	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "apply-") {
			t.Errorf("unexpected apply job in plan-only mode: %s", jobName)
		}
	}

	// No apply stages should exist
	for _, stage := range pipeline.Stages {
		if strings.Contains(stage, "-apply-") {
			t.Errorf("unexpected apply stage in plan-only mode: %s", stage)
		}
	}

	// Plan jobs should depend on plan jobs (not apply)
	AssertJobHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-eks", "plan-platform-stage-eu-central-1-vpc")
	AssertJobNotHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-eks", "apply-platform-stage-eu-central-1-vpc")
}

// TestFixture_ChangedOnly tests changed-only mode with fixtures
func TestFixture_ChangedOnly(t *testing.T) {
	fixture := LoadFixture(t, "basic")

	// Simulate only EKS and App changed
	changedModules := []*discovery.Module{
		fixture.GetModuleByName("eks"),
		fixture.GetModuleByName("app"),
	}

	// Filter to only stage environment modules
	var stageChangedModules []*discovery.Module
	for _, m := range changedModules {
		if m != nil && m.Environment == "stage" {
			stageChangedModules = append(stageChangedModules, m)
		}
	}

	result, err := fixture.Generator.Generate(stageChangedModules)
	if err != nil {
		t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should only have jobs for EKS and App (2 plan + 2 apply = 4)
	AssertJobCount(t, pipeline, 4)

	// Jobs for changed modules should exist
	AssertJobExists(t, pipeline, "plan-platform-stage-eu-central-1-eks")
	AssertJobExists(t, pipeline, "apply-platform-stage-eu-central-1-eks")
	AssertJobExists(t, pipeline, "plan-platform-stage-eu-central-1-app")
	AssertJobExists(t, pipeline, "apply-platform-stage-eu-central-1-app")

	// Jobs for non-changed modules should not exist
	AssertJobNotExists(t, pipeline, "plan-platform-stage-eu-central-1-vpc")
	AssertJobNotExists(t, pipeline, "apply-platform-stage-eu-central-1-vpc")
	AssertJobNotExists(t, pipeline, "plan-platform-stage-eu-central-1-s3")
	AssertJobNotExists(t, pipeline, "apply-platform-stage-eu-central-1-s3")

	// EKS should NOT depend on VPC (VPC not in target modules)
	AssertJobNotHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-eks", "apply-platform-stage-eu-central-1-vpc")

	// App should depend on EKS (in target modules)
	AssertJobHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-app", "apply-platform-stage-eu-central-1-eks")

	// App should NOT depend on S3 or RDS (not in target modules)
	AssertJobNotHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-app", "apply-platform-stage-eu-central-1-s3")
	AssertJobNotHasNeed(t, pipeline, "plan-platform-stage-eu-central-1-app", "apply-platform-stage-eu-central-1-rds")
}

// TestFixture_ChangedOnlyPlanOnly tests combined changed-only and plan-only modes
func TestFixture_ChangedOnlyPlanOnly(t *testing.T) {
	fixture := LoadFixtureWithConfig(t, "basic", func(cfg *config.Config) {
		cfg.GitLab.PlanOnly = true
		cfg.GitLab.PlanEnabled = true
	})

	// Only EKS changed
	eksModule := fixture.GetModuleByName("eks")
	if eksModule == nil {
		t.Fatal("EKS module not found")
	}

	// Filter to stage environment
	var changedModules []*discovery.Module
	if eksModule.Environment == "stage" {
		changedModules = []*discovery.Module{eksModule}
	}

	result, err := fixture.Generator.Generate(changedModules)
	if err != nil {
		t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have only 1 plan job
	AssertJobCount(t, pipeline, 1)

	// Only EKS plan should exist
	AssertJobExists(t, pipeline, "plan-platform-stage-eu-central-1-eks")

	// No apply jobs
	for jobName := range pipeline.Jobs {
		if strings.HasPrefix(jobName, "apply-") {
			t.Errorf("unexpected apply job: %s", jobName)
		}
	}

	// EKS plan should have no needs (VPC not in target)
	eksNeeds := GetJobNeeds(pipeline, "plan-platform-stage-eu-central-1-eks")
	if len(eksNeeds) != 0 {
		t.Errorf("EKS plan should have no needs when VPC not in target, got %v", eksNeeds)
	}
}

// TestFixture_EnvironmentFilter tests filtering by environment
func TestFixture_EnvironmentFilter(t *testing.T) {
	fixture := LoadFixture(t, "basic")

	// Get only stage modules
	stageModules := fixture.GetModulesByEnvironment("stage")
	if len(stageModules) != 5 {
		t.Errorf("expected 5 stage modules, got %d", len(stageModules))
	}

	result, err := fixture.Generator.Generate(stageModules)
	if err != nil {
		t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have 10 jobs (5 plan + 5 apply for stage)
	AssertJobCount(t, pipeline, 10)

	// Prod VPC should not exist
	AssertJobNotExists(t, pipeline, "plan-platform-prod-eu-central-1-vpc")
	AssertJobNotExists(t, pipeline, "apply-platform-prod-eu-central-1-vpc")
}

// TestFixture_Submodules tests submodule support with fixtures
func TestFixture_Submodules(t *testing.T) {
	fixture := LoadFixture(t, "submodules")

	// Should discover 3 modules (base + ec2/web + ec2/worker)
	if len(fixture.Modules) != 3 {
		t.Errorf("expected 3 modules, got %d", len(fixture.Modules))
		for _, m := range fixture.Modules {
			t.Logf("  module: %s", m.ID())
		}
	}

	result, err := fixture.Generator.Generate(fixture.Modules)
	if err != nil {
		t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have 6 jobs (3 plan + 3 apply)
	AssertJobCount(t, pipeline, 6)

	// Check submodule jobs exist
	AssertJobExists(t, pipeline, "plan-svc-stage-eu-central-1-ec2-web")
	AssertJobExists(t, pipeline, "apply-svc-stage-eu-central-1-ec2-web")
	AssertJobExists(t, pipeline, "plan-svc-stage-eu-central-1-ec2-worker")
	AssertJobExists(t, pipeline, "apply-svc-stage-eu-central-1-ec2-worker")

	// Submodules should depend on base
	AssertJobHasNeed(t, pipeline, "plan-svc-stage-eu-central-1-ec2-web", "apply-svc-stage-eu-central-1-base")
	AssertJobHasNeed(t, pipeline, "plan-svc-stage-eu-central-1-ec2-worker", "apply-svc-stage-eu-central-1-base")
}

// TestFixture_CyclicDependencies tests cycle detection with fixtures
func TestFixture_CyclicDependencies(t *testing.T) {
	fixture := LoadFixture(t, "cyclic")

	// Should detect cycles
	cycles := fixture.DepGraph.DetectCycles()
	if len(cycles) == 0 {
		t.Error("expected to detect cycles, found none")
	}

	// Topological sort should fail
	_, err := fixture.DepGraph.TopologicalSort()
	if err == nil {
		t.Error("expected topological sort to fail with cycles")
	}
}

// TestFixture_ApplyDependsOnPlan tests that apply jobs depend on their plan jobs
func TestFixture_ApplyDependsOnPlan(t *testing.T) {
	fixture := LoadFixture(t, "basic")

	result, err := fixture.Generator.Generate(fixture.Modules)
	if err != nil {
		t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Each apply job should depend on its own plan job
	for _, module := range fixture.Modules {
		moduleID := strings.ReplaceAll(module.ID(), "/", "-")
		applyJobName := "apply-" + moduleID
		planJobName := "plan-" + moduleID

		AssertJobHasNeed(t, pipeline, applyJobName, planJobName)
	}
}

// TestFixture_NoPlanEnabled tests pipeline without plan stage
func TestFixture_NoPlanEnabled(t *testing.T) {
	fixture := LoadFixtureWithConfig(t, "basic", func(cfg *config.Config) {
		cfg.GitLab.PlanEnabled = false
	})

	result, err := fixture.Generator.Generate(fixture.Modules)
	if err != nil {
		t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Should have only apply jobs (6 modules = 6 apply jobs)
	AssertJobCount(t, pipeline, 6)

	// No plan jobs
	planCount := CountJobsByPrefix(pipeline, "plan-")
	if planCount != 0 {
		t.Errorf("expected 0 plan jobs, got %d", planCount)
	}

	// No plan stages
	for _, stage := range pipeline.Stages {
		if strings.Contains(stage, "-plan-") {
			t.Errorf("unexpected plan stage: %s", stage)
		}
	}
}

// TestFixture_StageOrder tests that stages are in correct dependency order
func TestFixture_StageOrder(t *testing.T) {
	fixture := LoadFixture(t, "basic")

	result, err := fixture.Generator.Generate(fixture.GetModulesByEnvironment("stage"))
	if err != nil {
		t.Fatalf("failed to generate pipeline: %v", err)
	}

	pipeline, ok := result.(*gitlab.Pipeline)
	if !ok {
		t.Fatal("expected *gitlab.Pipeline type")
	}
	// Get stage indices
	stageIndex := make(map[string]int)
	for i, stage := range pipeline.Stages {
		stageIndex[stage] = i
	}

	// VPC and S3 should be in level 0 (no dependencies)
	vpcJob := pipeline.Jobs["apply-platform-stage-eu-central-1-vpc"]
	s3Job := pipeline.Jobs["apply-platform-stage-eu-central-1-s3"]

	// EKS and RDS should be in level 1 (depend on VPC)
	eksJob := pipeline.Jobs["apply-platform-stage-eu-central-1-eks"]
	rdsJob := pipeline.Jobs["apply-platform-stage-eu-central-1-rds"]

	// App should be in level 2 (depends on EKS, RDS, S3)
	appJob := pipeline.Jobs["apply-platform-stage-eu-central-1-app"]

	// Verify stage order
	if stageIndex[vpcJob.Stage] >= stageIndex[eksJob.Stage] {
		t.Error("VPC should be in earlier stage than EKS")
	}

	if stageIndex[s3Job.Stage] >= stageIndex[appJob.Stage] {
		t.Error("S3 should be in earlier stage than App")
	}

	if stageIndex[eksJob.Stage] >= stageIndex[appJob.Stage] {
		t.Error("EKS should be in earlier stage than App")
	}

	if stageIndex[rdsJob.Stage] >= stageIndex[appJob.Stage] {
		t.Error("RDS should be in earlier stage than App")
	}
}
