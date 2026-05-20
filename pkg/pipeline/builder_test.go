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

	ir, err := Build(testBuildOptions(modules, nil, BuildRequirements{}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	planJob := findJob(ir.Jobs, JobName(JobKindPlan, mod))
	if planJob == nil {
		t.Fatal("missing plan job")
	}
	applyJob := findJob(ir.Jobs, JobName(JobKindApply, mod))
	if applyJob == nil {
		t.Fatal("missing apply job")
	}
	if planJob.Kind != JobKindPlan {
		t.Fatalf("plan kind = %q", planJob.Kind)
	}
	if applyJob.Kind != JobKindApply {
		t.Fatalf("apply kind = %q", applyJob.Kind)
	}
}

func TestBuild_RequirementsPlanOnlySuppressesApply(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}

	ir, err := Build(testBuildOptions(modules, nil, BuildRequirements{PlanOnly: true}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if findJob(ir.Jobs, JobName(JobKindApply, mod)) != nil {
		t.Fatal("PlanOnly requirement should suppress apply jobs")
	}
}

func TestBuild_RequiredPlanJSONMakesOnlyMatchingModuleDetailed(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("svc", "prod", "eu", "vpc")
	app := discovery.TestModule("svc", "prod", "eu", "app")
	modules := []*discovery.Module{vpc, app}

	ir, err := Build(testBuildOptions(modules, [][2]int{{1, 0}}, RequirementsForResources(
		ModulePlanResource(ResourceKindPlanJSON, app.RelativePath),
	)))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	vpcPlan := findJob(ir.Jobs, JobName(JobKindPlan, vpc))
	appPlan := findJob(ir.Jobs, JobName(JobKindPlan, app))
	if vpcPlan.Operation.Terraform.DetailedPlan {
		t.Fatal("unrequested module plan should not be detailed")
	}
	if !appPlan.Operation.Terraform.DetailedPlan {
		t.Fatal("requested module plan should be detailed")
	}
	if !slices.Equal(appPlan.OutputArtifact.Paths, []string{PlanBinaryPath(app.RelativePath), PlanJSONPath(app.RelativePath)}) {
		t.Fatalf("app artifact paths = %v", appPlan.OutputArtifact.Paths)
	}
}

func TestBuild_ContributedPlanConsumerAddsArtifactDependency(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testBuildOptions(modules, nil, BuildRequirements{})
	opts.Contributions = []*Contribution{mustContribution(t, mustContributedJob(t, ContributedJobOptions{
		Name:     "cost-estimation",
		Commands: []string{"terraci cost"},
		Consumes: []ResourceRequest{AllPlanResources(ResourceKindPlanJSON)},
	}))}

	ir, err := Build(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	job := findJob(ir.Jobs, "cost-estimation")
	if job == nil {
		t.Fatal("cost-estimation job not found")
	}
	planName := JobName(JobKindPlan, mod)
	if !hasInputArtifact(job.InputArtifacts, PlanArtifactName(planName), planName, false) {
		t.Fatalf("input artifacts = %#v, want plan artifact", job.InputArtifacts)
	}
	if !hasDependency(job.Dependencies, planName) {
		t.Fatalf("dependencies = %#v, want artifact dependency on %s", job.Dependencies, planName)
	}
	if !findJob(ir.Jobs, planName).Operation.Terraform.DetailedPlan {
		t.Fatal("PlanJSON consumer should make plan detailed")
	}
}

func TestBuild_ApplyConsumesOnlyOwnPlanBinary(t *testing.T) {
	t.Parallel()

	vpc := discovery.TestModule("svc", "prod", "eu", "vpc")
	app := discovery.TestModule("svc", "prod", "eu", "app")
	modules := []*discovery.Module{vpc, app}

	ir, err := Build(testBuildOptions(modules, [][2]int{{1, 0}}, BuildRequirements{}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	appApply := findJob(ir.Jobs, JobName(JobKindApply, app))
	if appApply == nil {
		t.Fatal("app apply job not found")
	}
	if !hasDependency(appApply.Dependencies, JobName(JobKindPlan, app)) {
		t.Fatalf("app apply dependencies = %#v, want own plan artifact", appApply.Dependencies)
	}
	if !hasDependency(appApply.Dependencies, JobName(JobKindApply, vpc)) {
		t.Fatalf("app apply dependencies = %#v, want upstream apply control dep", appApply.Dependencies)
	}
	if !hasInputArtifact(appApply.InputArtifacts, PlanArtifactName(JobName(JobKindPlan, app)), JobName(JobKindPlan, app), false) {
		t.Fatalf("apply input artifacts = %#v, want own plan artifact only", appApply.InputArtifacts)
	}
}

func TestBuild_SummaryConsumesProducedReportsOnly(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testBuildOptions(modules, nil, BuildRequirements{})
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

	ir, err := Build(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	summary := findJob(ir.Jobs, "summary")
	if summary == nil {
		t.Fatal("summary job not found")
	}
	if !hasDependency(summary.Dependencies, "policy-check") {
		t.Fatalf("summary dependencies = %#v, want policy artifact dependency", summary.Dependencies)
	}
	if !hasDependency(summary.Dependencies, JobName(JobKindPlan, mod)) {
		t.Fatalf("summary dependencies = %#v, want plan artifact dependency", summary.Dependencies)
	}
	if !hasInputArtifact(summary.InputArtifacts, ResultArtifactName("policy-check"), "policy-check", true) {
		t.Fatalf("summary input artifacts = %#v, want optional policy artifact", summary.InputArtifacts)
	}
}

func TestBuild_RequiredPlanResourceWithPlanDisabledReturnsError(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testBuildOptions(modules, nil, RequirementsForResources(AllPlanResources(ResourceKindPlanJSON)))
	opts.PlanEnabled = false
	opts.Script.PlanEnabled = false

	_, err := Build(opts)
	if err == nil {
		t.Fatal("Build() error = nil, want missing resource error")
	}
	if !strings.Contains(err.Error(), "pipeline required resources requires unavailable plan_json for all modules") {
		t.Fatalf("Build() error = %q", err)
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

			opts := testBuildOptions(modules, nil, tt.requirements)
			opts.Contributions = tt.contributions
			_, err := Build(opts)
			if err == nil {
				t.Fatal("Build() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("Build() error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

func TestBuild_OptionalMissingPluginResourceDoesNotCreateDependency(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testBuildOptions(modules, nil, BuildRequirements{PlanOnly: true})
	opts.Contributions = []*Contribution{mustContribution(t, mustContributedJob(t, ContributedJobOptions{
		Name:     "summary",
		Commands: []string{"summary"},
		Consumes: []ResourceRequest{
			AllPluginResources(ResourceKindPluginReport, true),
		},
	}))}

	ir, err := Build(opts)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	summary := findJob(ir.Jobs, "summary")
	if summary == nil {
		t.Fatal("summary job not found")
	}
	if len(summary.Dependencies) != 0 {
		t.Fatalf("summary dependencies = %#v, want none", summary.Dependencies)
	}
	if len(summary.InputArtifacts) != 0 {
		t.Fatalf("summary input artifacts = %#v, want none", summary.InputArtifacts)
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

			opts := testBuildOptions(modules, nil, BuildRequirements{})
			opts.Contributions = []*Contribution{tt.contribution}
			_, err := Build(opts)
			if err == nil {
				t.Fatal("Build() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("Build() error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

func TestIR_ModuleCountCountsDistinctModules(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	ir := &IR{
		Jobs: []Job{
			{Name: "plan-svc-prod-eu-vpc", Kind: JobKindPlan, Module: mod},
			{Name: "apply-svc-prod-eu-vpc", Kind: JobKindApply, Module: mod},
			{Name: "summary", Kind: JobKindCommand},
		},
	}

	if got := ir.ModuleCount(); got != 1 {
		t.Fatalf("ModuleCount() = %d, want 1", got)
	}
}

func testBuildOptions(modules []*discovery.Module, edges [][2]int, requirements BuildRequirements) BuildOptions {
	return BuildOptions{
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
		if jobs[i].Name == name {
			return &jobs[i]
		}
	}
	return nil
}
