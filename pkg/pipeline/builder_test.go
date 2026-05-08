package pipeline

import (
	"slices"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
)

func TestBuild_SingleModule(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}

	ir, err := Build(testBuildOptions(modules, nil, BuildRequirements{}))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	mj := ir.Levels[0].Modules[0]
	if mj.Plan == nil {
		t.Fatal("missing plan job")
	}
	if mj.Apply == nil {
		t.Fatal("missing apply job")
	}
	if mj.Plan.Name != JobName(JobKindPlan, mod) {
		t.Fatalf("plan name = %q", mj.Plan.Name)
	}
	if mj.Apply.Name != JobName(JobKindApply, mod) {
		t.Fatalf("apply name = %q", mj.Apply.Name)
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
	if ir.Levels[0].Modules[0].Apply != nil {
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

	vpcPlan := ir.Levels[0].Modules[0].Plan
	appPlan := ir.Levels[1].Modules[0].Plan
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
	opts.Contributions = []*Contribution{{
		Jobs: []ContributedJob{{
			Name:     "cost-estimation",
			Commands: []string{"terraci cost"},
			Consumes: []ResourceRequest{AllPlanResources(ResourceKindPlanJSON)},
		}},
	}}

	ir, err := Build(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	job := ir.Jobs[0]
	planName := JobName(JobKindPlan, mod)
	if len(job.InputArtifacts) != 1 || job.InputArtifacts[0].Name != PlanArtifactName(planName) {
		t.Fatalf("input artifacts = %#v, want plan artifact", job.InputArtifacts)
	}
	if !hasDependency(job.Dependencies, planName, true) {
		t.Fatalf("dependencies = %#v, want artifact dependency on %s", job.Dependencies, planName)
	}
	if !ir.Levels[0].Modules[0].Plan.Operation.Terraform.DetailedPlan {
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

	appApply := ir.Levels[1].Modules[0].Apply
	if !hasDependency(appApply.Dependencies, JobName(JobKindPlan, app), true) {
		t.Fatalf("app apply dependencies = %#v, want own plan artifact", appApply.Dependencies)
	}
	if !hasDependency(appApply.Dependencies, JobName(JobKindApply, vpc), false) {
		t.Fatalf("app apply dependencies = %#v, want upstream apply control dep", appApply.Dependencies)
	}
	if len(appApply.InputArtifacts) != 1 || appApply.InputArtifacts[0].Name != PlanArtifactName(JobName(JobKindPlan, app)) {
		t.Fatalf("apply input artifacts = %#v, want own plan artifact only", appApply.InputArtifacts)
	}
}

func TestBuild_SummaryConsumesProducedReportsOnly(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	opts := testBuildOptions(modules, nil, BuildRequirements{})
	opts.Contributions = []*Contribution{{
		Jobs: []ContributedJob{
			{
				Name:     "policy-check",
				Commands: []string{"policy"},
				Produces: []ResourceSpec{
					PluginResource(ResourceKindPluginReport, "policy", ".terraci/policy-report.json"),
				},
			},
			{
				Name:     "summary",
				Commands: []string{"summary"},
				Consumes: []ResourceRequest{
					AllPlanResources(ResourceKindPlanJSON),
					AllPluginResources(ResourceKindPluginReport, true),
				},
			},
		},
	}}

	ir, err := Build(opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	summary := findJob(ir.Jobs, "summary")
	if summary == nil {
		t.Fatal("summary job not found")
	}
	if !hasDependency(summary.Dependencies, "policy-check", true) {
		t.Fatalf("summary dependencies = %#v, want policy artifact dependency", summary.Dependencies)
	}
	if !hasDependency(summary.Dependencies, JobName(JobKindPlan, mod), true) {
		t.Fatalf("summary dependencies = %#v, want plan artifact dependency", summary.Dependencies)
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

func TestBuild_RejectsInvalidContributedJobGraph(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}

	tests := []struct {
		name          string
		jobs          []ContributedJob
		wantErrSubstr string
	}{
		{
			name:          "unnamed contributed job",
			jobs:          []ContributedJob{{Commands: []string{"check"}}},
			wantErrSubstr: "unnamed job",
		},
		{
			name: "duplicate contributed job",
			jobs: []ContributedJob{
				{Name: "check", Commands: []string{"check"}},
				{Name: "check", Commands: []string{"check"}},
			},
			wantErrSubstr: `duplicate job name "check"`,
		},
		{
			name: "contributed job collides with module job",
			jobs: []ContributedJob{{
				Name: JobName(JobKindPlan, mod), Commands: []string{"check"},
			}},
			wantErrSubstr: `duplicate job name "` + JobName(JobKindPlan, mod) + `"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := testBuildOptions(modules, nil, BuildRequirements{})
			opts.Contributions = []*Contribution{{Jobs: tt.jobs}}
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

func TestIR_JobRefs(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	ir := &IR{
		Levels: []Level{{
			Index: 2,
			Modules: []ModuleJobs{{
				Module: mod,
				Plan:   &Job{Name: "plan-svc-prod-eu-vpc"},
				Apply:  &Job{Name: "apply-svc-prod-eu-vpc"},
			}},
		}},
		Jobs: []Job{{Name: "summary"}},
	}

	refs := ir.JobRefs()
	if len(refs) != 3 {
		t.Fatalf("JobRefs() len = %d, want 3", len(refs))
	}
	if refs[0].Kind != JobKindPlan || refs[0].Level != 2 || refs[0].Module != mod {
		t.Fatalf("refs[0] = %#v, want level 2 plan module ref", refs[0])
	}
	if refs[1].Kind != JobKindApply || refs[1].Level != 2 || refs[1].Module != mod {
		t.Fatalf("refs[1] = %#v, want level 2 apply module ref", refs[1])
	}
	if refs[2].Kind != JobKindContributed || refs[2].Job.Name != "summary" {
		t.Fatalf("refs[2] = %#v, want contributed summary", refs[2])
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

func hasDependency(deps []JobDependency, name string, artifacts bool) bool {
	for _, dep := range deps {
		if dep.Job == name && dep.Artifacts == artifacts {
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
