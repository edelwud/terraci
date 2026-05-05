package pipeline

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
)

func TestBuild_SingleModule(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	depGraph := buildGraph(modules, nil)
	index := discovery.NewModuleIndex(modules)

	ir, err := Build(BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script:        ScriptConfig{PlanEnabled: true},
		PlanEnabled:   true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(ir.Levels) != 1 {
		t.Fatalf("expected 1 level, got %d", len(ir.Levels))
	}
	if len(ir.Levels[0].Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(ir.Levels[0].Modules))
	}

	mj := ir.Levels[0].Modules[0]
	if mj.Plan == nil {
		t.Error("missing plan job")
	}
	if mj.Apply == nil {
		t.Error("missing apply job")
	}
	if mj.Plan.Name != JobName(JobKindPlan, mod) {
		t.Errorf("plan name = %q, want %q", mj.Plan.Name, JobName(JobKindPlan, mod))
	}
	if mj.Apply.Name != JobName(JobKindApply, mod) {
		t.Errorf("apply name = %q, want %q", mj.Apply.Name, JobName(JobKindApply, mod))
	}
}

func TestBuild_PlanOnly(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	depGraph := buildGraph(modules, nil)
	index := discovery.NewModuleIndex(modules)

	ir, err := Build(BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script:        ScriptConfig{PlanEnabled: true},
		PlanEnabled:   true,
		PlanOnly:      true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	for _, level := range ir.Levels {
		for _, mj := range level.Modules {
			if mj.Apply != nil {
				t.Error("plan-only should have no apply jobs")
			}
			if mj.Plan == nil {
				t.Error("plan-only should still have plan jobs")
			}
		}
	}
}

func TestBuild_PlanDisabled(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	depGraph := buildGraph(modules, nil)
	index := discovery.NewModuleIndex(modules)

	ir, err := Build(BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script:        ScriptConfig{},
		PlanEnabled:   false,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	for _, level := range ir.Levels {
		for _, mj := range level.Modules {
			if mj.Plan != nil {
				t.Error("plan-disabled should have no plan jobs")
			}
			if mj.Apply == nil {
				t.Error("plan-disabled should still have apply jobs")
			}
		}
	}
}

func TestBuild_WithSteps(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	depGraph := buildGraph(modules, nil)
	index := discovery.NewModuleIndex(modules)

	contributions := []*Contribution{{
		Steps: []Step{
			{Phase: PhasePrePlan, Name: "lint", Command: "tflint"},
			{Phase: PhasePostApply, Name: "notify", Command: "notify-slack"},
		},
	}}

	ir, err := Build(BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script:        ScriptConfig{PlanEnabled: true},
		Contributions: contributions,
		PlanEnabled:   true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	mj := ir.Levels[0].Modules[0]

	// Plan job should have PrePlan step
	hasPrePlan := false
	for _, s := range mj.Plan.Steps {
		if s.Phase == PhasePrePlan && s.Command == "tflint" {
			hasPrePlan = true
		}
	}
	if !hasPrePlan {
		t.Error("plan job missing PrePlan step")
	}

	// Apply job should have PostApply step
	hasPostApply := false
	for _, s := range mj.Apply.Steps {
		if s.Phase == PhasePostApply && s.Command == "notify-slack" {
			hasPostApply = true
		}
	}
	if !hasPostApply {
		t.Error("apply job missing PostApply step")
	}
}

func TestBuild_ContributedJobDeps(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	depGraph := buildGraph(modules, nil)
	index := discovery.NewModuleIndex(modules)

	contributions := []*Contribution{{
		Jobs: []ContributedJob{{
			Name: "check", Phase: PhasePostPlan,
			Commands: []string{"check"}, DependsOnPlan: true,
		}},
	}}

	ir, err := Build(BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script:        ScriptConfig{PlanEnabled: true},
		Contributions: contributions,
		PlanEnabled:   true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(ir.Jobs) != 1 {
		t.Fatalf("expected 1 contributed job, got %d", len(ir.Jobs))
	}

	// Job should depend on plan job
	planName := JobName(JobKindPlan, mod)
	hasDep := false
	for _, d := range ir.Jobs[0].Dependencies {
		if d == planName {
			hasDep = true
		}
	}
	if !hasDep {
		t.Errorf("contributed job should depend on %s", planName)
	}
}

func TestBuild_FinalizeJobDependsOnOtherContributed(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	depGraph := buildGraph(modules, nil)
	index := discovery.NewModuleIndex(modules)

	contributions := []*Contribution{{
		Jobs: []ContributedJob{
			{Name: "policy-check", Phase: PhasePostPlan, Commands: []string{"check"}, DependsOnPlan: true},
			{Name: "summary", Phase: PhaseFinalize, Commands: []string{"summarize"}, DependsOnPlan: true},
		},
	}}

	ir, err := Build(BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script:        ScriptConfig{PlanEnabled: true},
		Contributions: contributions,
		PlanEnabled:   true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(ir.Jobs) != 2 {
		t.Fatalf("expected 2 contributed jobs, got %d", len(ir.Jobs))
	}

	// Find the finalize job
	var finalizeJob *Job
	for i := range ir.Jobs {
		if ir.Jobs[i].Name == "summary" {
			finalizeJob = &ir.Jobs[i]
			break
		}
	}
	if finalizeJob == nil {
		t.Fatal("finalize job not found")
	}

	// Finalize should depend on policy-check
	hasPolicyDep := false
	for _, d := range finalizeJob.Dependencies {
		if d == "policy-check" {
			hasPolicyDep = true
		}
	}
	if !hasPolicyDep {
		t.Error("finalize job should depend on policy-check")
	}
}

func TestBuild_MultipleFinalizeJobsDoNotImplicitlyCycle(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	depGraph := buildGraph(modules, nil)
	index := discovery.NewModuleIndex(modules)

	contributions := []*Contribution{{
		Jobs: []ContributedJob{
			{Name: "policy-check", Phase: PhasePostPlan, Commands: []string{"check"}, DependsOnPlan: true},
			{Name: "summary", Phase: PhaseFinalize, Commands: []string{"summarize"}, DependsOnPlan: true},
			{Name: "notify", Phase: PhaseFinalize, Commands: []string{"notify"}, DependsOnPlan: true},
		},
	}}

	ir, err := Build(BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script:        ScriptConfig{PlanEnabled: true},
		Contributions: contributions,
		PlanEnabled:   true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	for i := range ir.Jobs {
		job := &ir.Jobs[i]
		if job.Phase != PhaseFinalize {
			continue
		}
		for _, dep := range job.Dependencies {
			if dep == "summary" || dep == "notify" {
				t.Fatalf("finalize job %q has implicit finalize dependency %q", job.Name, dep)
			}
		}
	}
}

func TestBuild_RejectsInvalidContributedJobGraph(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	depGraph := buildGraph(modules, nil)
	index := discovery.NewModuleIndex(modules)

	tests := []struct {
		name          string
		contribution  *Contribution
		wantErrSubstr string
	}{
		{
			name: "unnamed contributed job",
			contribution: &Contribution{Jobs: []ContributedJob{{
				Phase: PhasePostPlan, Commands: []string{"check"},
			}}},
			wantErrSubstr: "unnamed job",
		},
		{
			name: "duplicate contributed job",
			contribution: &Contribution{Jobs: []ContributedJob{
				{Name: "check", Phase: PhasePostPlan, Commands: []string{"check"}},
				{Name: "check", Phase: PhaseFinalize, Commands: []string{"check"}},
			}},
			wantErrSubstr: `duplicate job name "check"`,
		},
		{
			name: "contributed job collides with module job",
			contribution: &Contribution{Jobs: []ContributedJob{{
				Name: JobName(JobKindPlan, mod), Phase: PhasePostPlan, Commands: []string{"check"},
			}}},
			wantErrSubstr: `duplicate job name "` + JobName(JobKindPlan, mod) + `"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Build(BuildOptions{
				DepGraph:      depGraph,
				TargetModules: modules,
				AllModules:    modules,
				ModuleIndex:   index,
				Script:        ScriptConfig{PlanEnabled: true},
				Contributions: []*Contribution{tt.contribution},
				PlanEnabled:   true,
			})
			if err == nil {
				t.Fatal("Build() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("Build() error = %q, want substring %q", err.Error(), tt.wantErrSubstr)
			}
		})
	}
}

func TestBuild_MultipleModulesWithDependencies(t *testing.T) {
	t.Parallel()

	modA := discovery.TestModule("svc", "prod", "eu", "vpc")
	modB := discovery.TestModule("svc", "prod", "eu", "rds")
	modules := []*discovery.Module{modA, modB}
	depGraph := buildGraph(modules, [][2]int{{1, 0}}) // B depends on A
	index := discovery.NewModuleIndex(modules)

	ir, err := Build(BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script:        ScriptConfig{PlanEnabled: true},
		PlanEnabled:   true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Should have 2 levels since B depends on A
	if len(ir.Levels) != 2 {
		t.Fatalf("expected 2 levels, got %d", len(ir.Levels))
	}

	// Check planNames
	planNames := ir.planNames()
	if len(planNames) != 2 {
		t.Fatalf("expected 2 plan names, got %d", len(planNames))
	}
}

func TestBuild_NoContributions(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "vpc")
	modules := []*discovery.Module{mod}
	depGraph := buildGraph(modules, nil)
	index := discovery.NewModuleIndex(modules)

	ir, err := Build(BuildOptions{
		DepGraph:      depGraph,
		TargetModules: modules,
		AllModules:    modules,
		ModuleIndex:   index,
		Script:        ScriptConfig{PlanEnabled: true},
		PlanEnabled:   true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(ir.Jobs) != 0 {
		t.Errorf("expected 0 contributed jobs, got %d", len(ir.Jobs))
	}
}

func TestIR_PlanNames_Empty(t *testing.T) {
	t.Parallel()

	ir := &IR{}
	names := ir.planNames()
	if len(names) != 0 {
		t.Errorf("expected 0 plan names, got %d", len(names))
	}
}

func TestIR_ContributedJobNames(t *testing.T) {
	t.Parallel()

	ir := &IR{
		Jobs: []Job{
			{Name: "policy-check"},
			{Name: "summary"},
		},
	}
	names := ir.contributedJobNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "policy-check" {
		t.Errorf("names[0] = %q, want policy-check", names[0])
	}
	if names[1] != "summary" {
		t.Errorf("names[1] = %q, want summary", names[1])
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
		Jobs: []Job{{Name: "summary", Phase: PhaseFinalize}},
	}

	refs := ir.JobRefs()
	if len(refs) != 3 {
		t.Fatalf("JobRefs() len = %d, want 3", len(refs))
	}
	if refs[0].Kind != JobKindContributed || refs[0].Job.Name != "summary" {
		t.Fatalf("refs[0] = %#v, want contributed summary", refs[0])
	}
	if refs[1].Kind != JobKindPlan || refs[1].Level != 2 || refs[1].Module != mod {
		t.Fatalf("refs[1] = %#v, want level 2 plan module ref", refs[1])
	}
	if refs[2].Kind != JobKindApply || refs[2].Level != 2 || refs[2].Module != mod {
		t.Fatalf("refs[2] = %#v, want level 2 apply module ref", refs[2])
	}

	names := ir.jobNames()
	wantNames := []string{"summary", "plan-svc-prod-eu-vpc", "apply-svc-prod-eu-vpc"}
	for i := range wantNames {
		if names[i] != wantNames[i] {
			t.Fatalf("jobNames()[%d] = %q, want %q", i, names[i], wantNames[i])
		}
	}
}

func TestPhase_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		phase Phase
		want  string
	}{
		{PhasePrePlan, "pre-plan"},
		{PhasePostPlan, "post-plan"},
		{PhasePreApply, "pre-apply"},
		{PhasePostApply, "post-apply"},
		{PhaseFinalize, "finalize"},
		// Phase is a string type; ad-hoc values pass through verbatim.
		{Phase("custom"), "custom"},
	}

	for _, tt := range tests {
		if got := tt.phase.String(); got != tt.want {
			t.Errorf("Phase(%q).String() = %q, want %q", string(tt.phase), got, tt.want)
		}
	}
}

func TestFilterSteps(t *testing.T) {
	t.Parallel()

	steps := []Step{
		{Phase: PhasePrePlan, Name: "lint", Command: "tflint"},
		{Phase: PhasePostPlan, Name: "check", Command: "check"},
		{Phase: PhasePreApply, Name: "validate", Command: "validate"},
		{Phase: PhasePostApply, Name: "notify", Command: "notify"},
	}

	prePlan := filterSteps(steps, PhasePrePlan)
	if len(prePlan) != 1 || prePlan[0].Name != "lint" {
		t.Errorf("filterSteps(PrePlan) = %v, want [lint]", prePlan)
	}

	planPhases := filterSteps(steps, PhasePrePlan, PhasePostPlan)
	if len(planPhases) != 2 {
		t.Errorf("filterSteps(PrePlan, PostPlan) count = %d, want 2", len(planPhases))
	}

	empty := filterSteps(steps, PhaseFinalize)
	if len(empty) != 0 {
		t.Errorf("filterSteps(Finalize) count = %d, want 0", len(empty))
	}

	noSteps := filterSteps(nil, PhasePrePlan)
	if len(noSteps) != 0 {
		t.Errorf("filterSteps(nil) count = %d, want 0", len(noSteps))
	}
}

func TestHasContributedJobs(t *testing.T) {
	t.Parallel()

	if hasContributedJobs(nil) {
		t.Error("nil contributions should return false")
	}
	if hasContributedJobs([]*Contribution{}) {
		t.Error("empty contributions should return false")
	}
	if hasContributedJobs([]*Contribution{{Steps: []Step{{Name: "x"}}}}) {
		t.Error("contributions with only steps should return false")
	}
	if !hasContributedJobs([]*Contribution{{Jobs: []ContributedJob{{Name: "x"}}}}) {
		t.Error("contributions with jobs should return true")
	}
}
