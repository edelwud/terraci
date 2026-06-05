package pipeline

import (
	"slices"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
)

func mustContribution(tb testing.TB, jobs ...ContributedJob) *Contribution {
	tb.Helper()
	contribution, err := NewContribution(jobs...)
	if err != nil {
		tb.Fatalf("NewContribution() error = %v", err)
	}
	return contribution
}

func mustContributionSet(tb testing.TB, contributions ...*Contribution) ContributionSet {
	tb.Helper()
	set, err := NewContributionSet(contributions...)
	if err != nil {
		tb.Fatalf("NewContributionSet() error = %v", err)
	}
	return set
}

func mustContributedJob(tb testing.TB, opts ContributedJobOptions) ContributedJob {
	tb.Helper()
	job, err := NewContributedJob(opts)
	if err != nil {
		tb.Fatalf("NewContributedJob() error = %v", err)
	}
	return job
}

func TestBuild_SingleModule(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}

	ir, err := buildProjectIR(testProjectIRBuildInput(modules, nil, mustIntent(t, true)))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	planJob := findJob(ir.jobs, jobName(JobKindPlan, mod))
	if planJob == nil {
		t.Fatal("missing plan job")
	}
	applyJob := findJob(ir.jobs, jobName(JobKindApply, mod))
	if applyJob == nil {
		t.Fatal("missing apply job")
	}
	if planJob.kind != JobKindPlan {
		t.Fatalf("plan kind = %q", planJob.kind)
	}
	if applyJob.kind != JobKindApply {
		t.Fatalf("apply kind = %q", applyJob.kind)
	}
}

func TestBuild_PlanIntentSuppressesApply(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}

	ir, err := buildProjectIR(testProjectIRBuildInput(modules, nil, mustIntent(t, false, AllPlanResources(ResourceKindPlanBinary))))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if findJob(ir.jobs, jobName(JobKindApply, mod)) != nil {
		t.Fatal("plan intent should suppress apply jobs")
	}
}

func TestBuild_RequiredPlanJSONMakesOnlyMatchingModuleDetailed(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("svc", "prod", "eu", "vpc")
	app := discovery.TestModule("svc", "prod", "eu", "app")
	modules := []*discovery.Module{vpc, app}

	ir, err := buildProjectIR(testProjectIRBuildInput(modules, [][2]int{{1, 0}}, mustIntent(t, true,
		ModulePlanResource(ResourceKindPlanJSON, app.RelativePath),
	)))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	vpcPlan := findJob(ir.jobs, jobName(JobKindPlan, vpc))
	appPlan := findJob(ir.jobs, jobName(JobKindPlan, app))
	if vpcPlan.operation.terraform.detailedPlan {
		t.Fatal("unrequested module plan should not be detailed")
	}
	if !appPlan.operation.terraform.detailedPlan {
		t.Fatal("requested module plan should be detailed")
	}
	if !slices.Equal(appPlan.outputArtifact.Paths, []string{PlanBinaryPath(app.RelativePath), PlanJSONPath(app.RelativePath)}) {
		t.Fatalf("app artifact paths = %v", appPlan.outputArtifact.Paths)
	}
}

func TestBuild_ContributedPlanConsumerAddsArtifactDependency(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testProjectIRBuildInput(modules, nil, mustIntent(t, true))
	opts.Contributions = mustContributionSet(t, mustContribution(t, mustContributedJob(t, ContributedJobOptions{
		Name:     "cost-estimation",
		Commands: []string{"terraci cost"},
		Consumes: []ResourceRequest{AllPlanResources(ResourceKindPlanJSON)},
	})))

	ir, err := buildProjectIR(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	job := findJob(ir.jobs, "cost-estimation")
	if job == nil {
		t.Fatal("cost-estimation job not found")
	}
	planName := jobName(JobKindPlan, mod)
	if !hasInputArtifact(job.inputArtifacts, PlanArtifactName(planName), planName, false) {
		t.Fatalf("input artifacts = %#v, want plan artifact", job.inputArtifacts)
	}
	if !hasDependency(job.dependencies, planName) {
		t.Fatalf("dependencies = %#v, want artifact dependency on %s", job.dependencies, planName)
	}
	if !findJob(ir.jobs, planName).operation.terraform.detailedPlan {
		t.Fatal("PlanJSON consumer should make plan detailed")
	}
}

func TestBuild_TerraformJobsMergeExecutionEnv(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testProjectIRBuildInput(modules, nil, mustIntent(t, true))
	opts.Terraform = mustTerraformJobConfigWithEnv(t, map[string]string{
		"TF_MODULE":        "override",
		"TF_IN_AUTOMATION": "true",
		"CUSTOM":           "value",
	})

	ir, err := buildProjectIR(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	planJob := findJob(ir.jobs, jobName(JobKindPlan, mod))
	if planJob == nil {
		t.Fatal("missing plan job")
	}
	env := planJob.Env()
	if env["TF_MODULE"] != "vpc" {
		t.Fatalf("TF_MODULE = %q, want module-derived value", env["TF_MODULE"])
	}
	if env["TF_IN_AUTOMATION"] != "true" {
		t.Fatalf("TF_IN_AUTOMATION = %q, want execution env", env["TF_IN_AUTOMATION"])
	}
	if env["CUSTOM"] != "value" {
		t.Fatalf("CUSTOM = %q, want execution env", env["CUSTOM"])
	}
}

func TestBuild_ApplyConsumesOnlyOwnPlanBinary(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("svc", "prod", "eu", "vpc")
	app := discovery.TestModule("svc", "prod", "eu", "app")
	modules := []*discovery.Module{vpc, app}

	ir, err := buildProjectIR(testProjectIRBuildInput(modules, [][2]int{{1, 0}}, mustIntent(t, true)))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	appApply := findJob(ir.jobs, jobName(JobKindApply, app))
	if appApply == nil {
		t.Fatal("app apply job not found")
	}
	if !hasDependency(appApply.dependencies, jobName(JobKindPlan, app)) {
		t.Fatalf("app apply dependencies = %#v, want own plan artifact", appApply.dependencies)
	}
	if !hasDependency(appApply.dependencies, jobName(JobKindApply, vpc)) {
		t.Fatalf("app apply dependencies = %#v, want upstream apply control dep", appApply.dependencies)
	}
	if !hasInputArtifact(appApply.inputArtifacts, PlanArtifactName(jobName(JobKindPlan, app)), jobName(JobKindPlan, app), false) {
		t.Fatalf("apply input artifacts = %#v, want own plan artifact only", appApply.inputArtifacts)
	}
}

func TestBuild_SummaryConsumesProducedReportsOnly(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testProjectIRBuildInput(modules, nil, mustIntent(t, true))
	opts.Contributions = mustContributionSet(t, mustContribution(t,
		mustContributedJob(t, ContributedJobOptions{
			Name:     "policy-check",
			Commands: []string{"policy"},
			Produces: []ResourceSpec{
				PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
			},
		}),
		mustContributedJob(t, ContributedJobOptions{
			Name:     "summary",
			Commands: []string{"summary"},
			Consumes: []ResourceRequest{
				AllPlanResources(ResourceKindPlanJSON),
				AllPluginResources(ResourceKindPluginReport, true),
			},
		}),
	))

	ir, err := buildProjectIR(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	summary := findJob(ir.jobs, "summary")
	if summary == nil {
		t.Fatal("summary job not found")
	}
	if !hasDependency(summary.dependencies, "policy-check") {
		t.Fatalf("summary dependencies = %#v, want policy artifact dependency", summary.dependencies)
	}
	if !hasDependency(summary.dependencies, jobName(JobKindPlan, mod)) {
		t.Fatalf("summary dependencies = %#v, want plan artifact dependency", summary.dependencies)
	}
	if !hasInputArtifact(summary.inputArtifacts, ResultArtifactName("policy-check"), "policy-check", true) {
		t.Fatalf("summary input artifacts = %#v, want optional policy artifact", summary.inputArtifacts)
	}
}

func TestBuild_RequiredPlanResourceCreatesPlanJobWhenApplyDisabled(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testProjectIRBuildInput(modules, nil, mustIntent(t, false, AllPlanResources(ResourceKindPlanJSON)))

	ir, err := buildProjectIR(opts)
	if err != nil {
		t.Fatalf("buildProjectIR() error = %v", err)
	}
	if findJob(ir.jobs, jobName(JobKindApply, mod)) != nil {
		t.Fatal("unexpected apply job")
	}
	planJob := findJob(ir.jobs, jobName(JobKindPlan, mod))
	if planJob == nil {
		t.Fatal("missing resource-driven plan job")
	}
	if !planJob.operation.terraform.detailedPlan {
		t.Fatal("plan_json request should make plan job detailed")
	}
}

func TestBuild_ValidatesResourceRequestsWithContext(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}

	tests := []struct {
		name          string
		intent        BuildIntent
		contributions ContributionSet
		wantErrSubstr string
	}{
		{
			name: "missing selector scope in requirements",
			intent: BuildIntent{
				constructed: true,
				resources: []ResourceRequest{{
					kind: ResourceKindPlanJSON,
				}},
			},
			wantErrSubstr: "requirements.resources[0]: plan_json selector scope is required",
		},
		{
			name: "plan resource cannot use producer selector",
			intent: BuildIntent{
				constructed: true,
				resources: []ResourceRequest{{
					kind: ResourceKindPlanJSON,
					selector: ResourceSelector{
						scope:    ResourceScopeProducer,
						producer: "cost",
					},
				}},
			},
			wantErrSubstr: `requirements.resources[0]: plan_json cannot use producer-scoped selector "producer"`,
		},
		{
			name:   "plugin resource cannot use module selector",
			intent: mustIntent(t, true),
			contributions: ContributionSet{contributions: []*Contribution{{
				jobs: []ContributedJob{{
					name:     "summary",
					commands: []string{"summary"},
					consumes: []ResourceRequest{{
						kind: ResourceKindPluginReport,
						selector: ResourceSelector{
							scope:      ResourceScopeModule,
							modulePath: mod.RelativePath,
						},
					}},
				}},
			}}},
			wantErrSubstr: `contributions[0].jobs[0].consumes[0]: plugin_report cannot use module-scoped selector "module"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testProjectIRBuildInput(modules, nil, tt.intent)
			opts.Contributions = tt.contributions
			_, err := buildProjectIR(opts)
			if err == nil {
				t.Fatal("buildProjectIR() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("buildProjectIR() error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

func TestBuild_OptionalMissingPluginResourceDoesNotCreateDependency(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testProjectIRBuildInput(modules, nil, mustIntent(t, false))
	opts.Contributions = mustContributionSet(t, mustContribution(t, mustContributedJob(t, ContributedJobOptions{
		Name:     "summary",
		Commands: []string{"summary"},
		Consumes: []ResourceRequest{
			AllPluginResources(ResourceKindPluginReport, true),
		},
	})))

	ir, err := buildProjectIR(opts)
	if err != nil {
		t.Fatalf("buildProjectIR() error = %v", err)
	}
	summary := findJob(ir.jobs, "summary")
	if summary == nil {
		t.Fatal("summary job not found")
	}
	if len(summary.dependencies) != 0 {
		t.Fatalf("summary dependencies = %#v, want none", summary.dependencies)
	}
	if len(summary.inputArtifacts) != 0 {
		t.Fatalf("summary input artifacts = %#v, want none", summary.inputArtifacts)
	}
}

func TestBuild_RejectsInvalidContributedJobGraph(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}

	tests := []struct {
		name          string
		contribution  *Contribution
		wantErrSubstr string
	}{
		{
			name: "contributed job collides with module job",
			contribution: mustContribution(t, mustContributedJob(t, ContributedJobOptions{
				Name:     jobName(JobKindPlan, mod),
				Commands: []string{"check"},
			})),
			wantErrSubstr: `duplicate job name "` + jobName(JobKindPlan, mod) + `"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testProjectIRBuildInput(modules, nil, mustIntent(t, true))
			opts.Contributions = mustContributionSet(t, tt.contribution)
			_, err := buildProjectIR(opts)
			if err == nil {
				t.Fatal("buildProjectIR() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("buildProjectIR() error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

func TestIR_ModuleCountCountsDistinctModules(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	ir := &IR{
		jobs: []Job{
			{name: "plan-svc-prod-eu-vpc", kind: JobKindPlan, module: mod},
			{name: "apply-svc-prod-eu-vpc", kind: JobKindApply, module: mod},
			{name: "summary", kind: JobKindCommand},
		},
	}

	if got := ir.ModuleCount(); got != 1 {
		t.Fatalf("ModuleCount() = %d, want 1", got)
	}
}

func mustIntent(tb testing.TB, applyEnabled bool, resources ...ResourceRequest) BuildIntent {
	tb.Helper()
	var (
		intent BuildIntent
		err    error
	)
	if applyEnabled {
		intent, err = ApplyBuildIntent(resources...)
	} else {
		intent, err = PlanBuildIntent(resources...)
	}
	if err != nil {
		tb.Fatalf("build intent error = %v", err)
	}
	return intent
}

func mustTerraformJobConfigWithEnv(tb testing.TB, env map[string]string) TerraformJobConfig {
	tb.Helper()
	cfg, err := NewTerraformJobConfig(TerraformJobConfigOptions{
		Binary:      "terraform",
		InitEnabled: true,
		Env:         env,
	})
	if err != nil {
		tb.Fatalf("NewTerraformJobConfig() error = %v", err)
	}
	return cfg
}

func testTerraformJobConfig() TerraformJobConfig {
	cfg, err := newTestTerraformJobConfig()
	if err != nil {
		panic(err)
	}
	return cfg
}

func newTestTerraformJobConfig() (TerraformJobConfig, error) {
	return NewTerraformJobConfig(TerraformJobConfigOptions{
		Binary:      "terraform",
		InitEnabled: true,
	})
}

func testProjectIRBuildInput(modules []*discovery.Module, edges [][2]int, intent BuildIntent) projectIRBuildInput {
	return projectIRBuildInput{
		DepGraph:      buildGraph(modules, edges),
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   discovery.NewModuleIndex(modules),
		Terraform:     testTerraformJobConfig(),
		Intent:        intent,
	}
}

func hasDependency(deps []JobDependency, name string) bool {
	for _, dep := range deps {
		if dep.Job == name {
			return true
		}
	}
	return false
}

func hasInputArtifact(inputs []InputArtifact, name, producer string, optional bool) bool {
	for _, input := range inputs {
		if input.Artifact.Name == name && input.ProducerJob == producer && input.Optional == optional {
			return true
		}
	}
	return false
}

func findJob(jobs []Job, name string) *Job {
	for i := range jobs {
		if jobs[i].name == name {
			return &jobs[i]
		}
	}
	return nil
}
