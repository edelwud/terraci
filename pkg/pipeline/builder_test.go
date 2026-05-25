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

	ir, err := buildProjectIR(testProjectIRBuildInput(modules, nil, BuildRequirements{}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	planJob := findJob(ir.jobs, JobName(JobKindPlan, mod))
	if planJob == nil {
		t.Fatal("missing plan job")
	}
	applyJob := findJob(ir.jobs, JobName(JobKindApply, mod))
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

func TestBuild_RequirementsPlanOnlySuppressesApply(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}

	ir, err := buildProjectIR(testProjectIRBuildInput(modules, nil, BuildRequirements{PlanOnly: true}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if findJob(ir.jobs, JobName(JobKindApply, mod)) != nil {
		t.Fatal("PlanOnly requirement should suppress apply jobs")
	}
}

func TestBuild_RequiredPlanJSONMakesOnlyMatchingModuleDetailed(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("svc", "prod", "eu", "vpc")
	app := discovery.TestModule("svc", "prod", "eu", "app")
	modules := []*discovery.Module{vpc, app}

	ir, err := buildProjectIR(testProjectIRBuildInput(modules, [][2]int{{1, 0}}, RequirementsForResources(
		ModulePlanResource(ResourceKindPlanJSON, app.RelativePath),
	)))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	vpcPlan := findJob(ir.jobs, JobName(JobKindPlan, vpc))
	appPlan := findJob(ir.jobs, JobName(JobKindPlan, app))
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
	opts := testProjectIRBuildInput(modules, nil, BuildRequirements{})
	opts.Contributions = []*Contribution{mustContribution(t, mustContributedJob(t, ContributedJobOptions{
		Name:     "cost-estimation",
		Commands: []string{"terraci cost"},
		Consumes: []ResourceRequest{AllPlanResources(ResourceKindPlanJSON)},
	}))}

	ir, err := buildProjectIR(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	job := findJob(ir.jobs, "cost-estimation")
	if job == nil {
		t.Fatal("cost-estimation job not found")
	}
	planName := JobName(JobKindPlan, mod)
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

func TestBuild_ApplyConsumesOnlyOwnPlanBinary(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("svc", "prod", "eu", "vpc")
	app := discovery.TestModule("svc", "prod", "eu", "app")
	modules := []*discovery.Module{vpc, app}

	ir, err := buildProjectIR(testProjectIRBuildInput(modules, [][2]int{{1, 0}}, BuildRequirements{}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	appApply := findJob(ir.jobs, JobName(JobKindApply, app))
	if appApply == nil {
		t.Fatal("app apply job not found")
	}
	if !hasDependency(appApply.dependencies, JobName(JobKindPlan, app)) {
		t.Fatalf("app apply dependencies = %#v, want own plan artifact", appApply.dependencies)
	}
	if !hasDependency(appApply.dependencies, JobName(JobKindApply, vpc)) {
		t.Fatalf("app apply dependencies = %#v, want upstream apply control dep", appApply.dependencies)
	}
	if !hasInputArtifact(appApply.inputArtifacts, PlanArtifactName(JobName(JobKindPlan, app)), JobName(JobKindPlan, app), false) {
		t.Fatalf("apply input artifacts = %#v, want own plan artifact only", appApply.inputArtifacts)
	}
}

func TestBuild_SummaryConsumesProducedReportsOnly(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testProjectIRBuildInput(modules, nil, BuildRequirements{})
	opts.Contributions = []*Contribution{mustContribution(t,
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
	)}

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
	if !hasDependency(summary.dependencies, JobName(JobKindPlan, mod)) {
		t.Fatalf("summary dependencies = %#v, want plan artifact dependency", summary.dependencies)
	}
	if !hasInputArtifact(summary.inputArtifacts, ResultArtifactName("policy-check"), "policy-check", true) {
		t.Fatalf("summary input artifacts = %#v, want optional policy artifact", summary.inputArtifacts)
	}
}

func TestBuild_RequiredPlanResourceWithPlanDisabledReturnsError(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testProjectIRBuildInput(modules, nil, RequirementsForResources(AllPlanResources(ResourceKindPlanJSON)))
	opts.PlanEnabled = false
	opts.Script.PlanEnabled = false

	_, err := buildProjectIR(opts)
	if err == nil {
		t.Fatal("buildProjectIR() error = nil, want missing resource error")
	}
	if !strings.Contains(err.Error(), "pipeline required resources requires unavailable plan_json for all modules") {
		t.Fatalf("buildProjectIR() error = %q", err)
	}
}

func TestBuild_ValidatesResourceRequestsWithContext(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}

	tests := []struct {
		name          string
		requirements  BuildRequirements
		contributions []*Contribution
		wantErrSubstr string
	}{
		{
			name: "missing selector scope in requirements",
			requirements: RequirementsForResources(ResourceRequest{
				Kind: ResourceKindPlanJSON,
			}),
			wantErrSubstr: "requirements.resources[0]: plan_json selector scope is required",
		},
		{
			name: "plan resource cannot use producer selector",
			requirements: RequirementsForResources(ResourceRequest{
				Kind: ResourceKindPlanJSON,
				Selector: ResourceSelector{
					Scope:    ResourceScopeProducer,
					Producer: "cost",
				},
			}),
			wantErrSubstr: `requirements.resources[0]: plan_json cannot use producer-scoped selector "producer"`,
		},
		{
			name: "plugin resource cannot use module selector",
			contributions: []*Contribution{{
				jobs: []ContributedJob{{
					name:     "summary",
					commands: []string{"summary"},
					consumes: []ResourceRequest{{
						Kind: ResourceKindPluginReport,
						Selector: ResourceSelector{
							Scope:      ResourceScopeModule,
							ModulePath: mod.RelativePath,
						},
					}},
				}},
			}},
			wantErrSubstr: `contributions[0].jobs[0].consumes[0]: plugin_report cannot use module-scoped selector "module"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testProjectIRBuildInput(modules, nil, tt.requirements)
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
	opts := testProjectIRBuildInput(modules, nil, BuildRequirements{PlanOnly: true})
	opts.Contributions = []*Contribution{mustContribution(t, mustContributedJob(t, ContributedJobOptions{
		Name:     "summary",
		Commands: []string{"summary"},
		Consumes: []ResourceRequest{
			AllPluginResources(ResourceKindPluginReport, true),
		},
	}))}

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
				Name:     JobName(JobKindPlan, mod),
				Commands: []string{"check"},
			})),
			wantErrSubstr: `duplicate job name "` + JobName(JobKindPlan, mod) + `"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testProjectIRBuildInput(modules, nil, BuildRequirements{})
			opts.Contributions = []*Contribution{tt.contribution}
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

func testProjectIRBuildInput(modules []*discovery.Module, edges [][2]int, requirements BuildRequirements) projectIRBuildInput {
	return projectIRBuildInput{
		DepGraph:      buildGraph(modules, edges),
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   discovery.NewModuleIndex(modules),
		Script:        ScriptConfig{PlanEnabled: true},
		Requirements:  requirements,
		PlanEnabled:   true,
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
