package pipeline

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
)

func buildGraph(modules []*discovery.Module, edges [][2]int) *graph.DependencyGraph {
	g := graph.NewDependencyGraph()
	for _, m := range modules {
		g.AddNode(m)
	}
	for _, e := range edges {
		g.AddEdge(modules[e[0]].ID(), modules[e[1]].ID())
	}
	return g
}

func TestBuildJobPlan(t *testing.T) {
	t.Parallel()

	modA := discovery.TestModule("svc", "prod", "us-east-1", "vpc")
	modB := discovery.TestModule("svc", "prod", "us-east-1", "rds")
	allModules := []*discovery.Module{modA, modB}
	idx := discovery.NewModuleIndex(allModules)
	depGraph := buildGraph(allModules, [][2]int{{1, 0}}) // B depends on A

	tests := []struct {
		name           string
		targets        []*discovery.Module
		hasJobs        bool
		planEnabled    bool
		wantCount      int
		wantContrib    bool
		wantLevelCount int
	}{
		{
			name:           "empty targets falls back to allModules",
			targets:        nil,
			hasJobs:        false,
			planEnabled:    false,
			wantCount:      2,
			wantContrib:    false,
			wantLevelCount: 2,
		},
		{
			name:           "non-empty targets used directly",
			targets:        []*discovery.Module{modA},
			hasJobs:        false,
			planEnabled:    false,
			wantCount:      1,
			wantContrib:    false,
			wantLevelCount: 1,
		},
		{
			name:        "contributed jobs and plan enabled set HasContributedJobs",
			targets:     allModules,
			hasJobs:     true,
			planEnabled: true,
			wantCount:   2,
			wantContrib: true,
		},
		{
			name:        "contributed jobs without plan do not affect dry run stage math",
			targets:     allModules,
			hasJobs:     true,
			planEnabled: false,
			wantCount:   2,
			wantContrib: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan, err := buildJobPlan(depGraph, tt.targets, allModules, idx, tt.hasJobs, tt.planEnabled)
			if err != nil {
				t.Fatalf("buildJobPlan() error = %v", err)
			}

			if got := len(plan.TargetModules); got != tt.wantCount {
				t.Errorf("TargetModules count = %d, want %d", got, tt.wantCount)
			}
			if plan.HasContributedJobs != tt.wantContrib {
				t.Errorf("HasContributedJobs = %v, want %v", plan.HasContributedJobs, tt.wantContrib)
			}
			if tt.wantLevelCount > 0 && len(plan.ExecutionLevels) != tt.wantLevelCount {
				t.Errorf("ExecutionLevels count = %d, want %d", len(plan.ExecutionLevels), tt.wantLevelCount)
			}
			if plan.Subgraph == nil {
				t.Error("Subgraph should not be nil")
			}
			for _, m := range plan.TargetModules {
				if plan.Subgraph.GetNode(m.ID()) == nil {
					t.Errorf("Subgraph missing target module %s", m.ID())
				}
			}
		})
	}
}

func TestJobName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		jobKind  JobKind
		module   *discovery.Module
		expected string
	}{
		{
			name:     "plan kind",
			jobKind:  JobKindPlan,
			module:   discovery.TestModule("svc", "prod", "us-east-1", "vpc"),
			expected: "plan-svc-prod-us-east-1-vpc",
		},
		{
			name:     "apply kind",
			jobKind:  JobKindApply,
			module:   discovery.TestModule("svc", "prod", "us-east-1", "vpc"),
			expected: "apply-svc-prod-us-east-1-vpc",
		},
		{
			name:     "contributed kind has no prefix",
			jobKind:  JobKindContributed,
			module:   discovery.TestModule("payments", "staging", "eu-west-1", "rds"),
			expected: "payments-staging-eu-west-1-rds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := JobName(tt.jobKind, tt.module)
			if got != tt.expected {
				t.Errorf("JobName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestResolveDependencyNames(t *testing.T) {
	t.Parallel()

	modA := discovery.TestModule("svc", "prod", "us-east-1", "vpc")
	modB := discovery.TestModule("svc", "prod", "us-east-1", "rds")
	modC := discovery.TestModule("svc", "prod", "us-east-1", "ecs")
	allModules := []*discovery.Module{modA, modB, modC}
	idx := discovery.NewModuleIndex(allModules)

	// B depends on A, C depends on A
	depGraph := buildGraph(allModules, [][2]int{{1, 0}, {2, 0}})

	tests := []struct {
		name        string
		module      *discovery.Module
		jobKind     JobKind
		targets     []*discovery.Module
		wantNames   []string
		wantNoNames bool
	}{
		{
			name:      "dep in target set returns job name",
			module:    modB,
			jobKind:   JobKindPlan,
			targets:   []*discovery.Module{modA, modB},
			wantNames: []string{"plan-svc-prod-us-east-1-vpc"},
		},
		{
			name:        "dep not in target set is excluded",
			module:      modB,
			jobKind:     JobKindPlan,
			targets:     []*discovery.Module{modB},
			wantNoNames: true,
		},
		{
			name:        "module with no dependencies returns empty",
			module:      modA,
			jobKind:     JobKindPlan,
			targets:     []*discovery.Module{modA, modB},
			wantNoNames: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ids := make([]string, len(tt.targets))
			for i, m := range tt.targets {
				ids[i] = m.ID()
			}
			subgraph := depGraph.Subgraph(ids)
			got := ResolveDependencyNames(tt.module, tt.jobKind, subgraph, idx)
			if tt.wantNoNames {
				if len(got) != 0 {
					t.Errorf("expected no names, got %v", got)
				}
				return
			}
			if len(got) != len(tt.wantNames) {
				t.Fatalf("got %d names, want %d: %v", len(got), len(tt.wantNames), got)
			}
			for i, want := range tt.wantNames {
				if got[i] != want {
					t.Errorf("name[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestResolveDependencyNames_DepNotInIndex(t *testing.T) {
	t.Parallel()

	modA := discovery.TestModule("svc", "prod", "us-east-1", "vpc")
	modB := discovery.TestModule("svc", "prod", "us-east-1", "rds")

	// Build graph with both modules
	depGraph := buildGraph([]*discovery.Module{modA, modB}, [][2]int{{1, 0}})

	// Build index with only modB (modA is missing from index)
	idx := discovery.NewModuleIndex([]*discovery.Module{modB})

	subgraph := depGraph.Subgraph([]string{modA.ID(), modB.ID()})
	got := ResolveDependencyNames(modB, JobKindPlan, subgraph, idx)
	// modA is in target set but not in index, so it should be skipped
	if len(got) != 0 {
		t.Errorf("expected no names when dep not in index, got %v", got)
	}
}

func TestIRDryRun(t *testing.T) {
	t.Parallel()

	modA := discovery.TestModule("svc", "prod", "us-east-1", "vpc")
	modB := discovery.TestModule("svc", "prod", "us-east-1", "rds")

	tests := []struct {
		name         string
		totalModules int
		wantJobs     int
		wantStages   int
		wantAffected int
		wantTotal    int
	}{
		{
			name:         "basic without contributed jobs",
			totalModules: 5,
			wantJobs:     2,
			wantStages:   2,
			wantAffected: 2,
			wantTotal:    5,
		},
		{
			name:         "planEnabled doubles job count per level",
			totalModules: 5,
			wantJobs:     4, // 2 levels * 1 module * 2 (plan+apply)
			wantStages:   2,
			wantAffected: 2,
			wantTotal:    5,
		},
		{
			name:         "contributed jobs add 1 job and 1 stage",
			totalModules: 3,
			wantJobs:     2, // 1 module + 1 contributed job
			wantStages:   2, // 1 level + 1 contributed stage
			wantAffected: 1,
			wantTotal:    3,
		},
		{
			name:         "multiple contributed jobs count individually and phases count once",
			totalModules: 10,
			wantJobs:     6, // 2*2 module jobs + 2 contributed jobs
			wantStages:   2, // 1 level + 1 contributed phase
			wantAffected: 2,
			wantTotal:    10,
		},
		{
			name:         "multiple contributed phases increase stage count",
			totalModules: 2,
			wantJobs:     3,
			wantStages:   3,
			wantAffected: 1,
			wantTotal:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var ir *IR
			switch tt.name {
			case "basic without contributed jobs":
				ir = &IR{
					Levels: []Level{
						{Index: 0, Modules: []ModuleJobs{{Module: modA, Apply: &Job{Name: "apply-a"}}}},
						{Index: 1, Modules: []ModuleJobs{{Module: modB, Apply: &Job{Name: "apply-b"}}}},
					},
				}
			case "planEnabled doubles job count per level":
				ir = &IR{
					Levels: []Level{
						{Index: 0, Modules: []ModuleJobs{{Module: modA, Plan: &Job{Name: "plan-a"}, Apply: &Job{Name: "apply-a"}}}},
						{Index: 1, Modules: []ModuleJobs{{Module: modB, Plan: &Job{Name: "plan-b"}, Apply: &Job{Name: "apply-b"}}}},
					},
				}
			case "contributed jobs add 1 job and 1 stage":
				ir = &IR{
					Levels: []Level{
						{Index: 0, Modules: []ModuleJobs{{Module: modA, Apply: &Job{Name: "apply-a"}}}},
					},
					Jobs: []Job{{Name: "summary", Phase: PhaseFinalize}},
				}
			case "multiple contributed jobs count individually and phases count once":
				ir = &IR{
					Levels: []Level{
						{Index: 0, Modules: []ModuleJobs{
							{Module: modA, Plan: &Job{Name: "plan-a"}, Apply: &Job{Name: "apply-a"}},
							{Module: modB, Plan: &Job{Name: "plan-b"}, Apply: &Job{Name: "apply-b"}},
						}},
					},
					Jobs: []Job{
						{Name: "policy", Phase: PhasePostPlan},
						{Name: "cost", Phase: PhasePostPlan},
					},
				}
			case "multiple contributed phases increase stage count":
				ir = &IR{
					Levels: []Level{
						{Index: 0, Modules: []ModuleJobs{{Module: modA, Apply: &Job{Name: "apply-a"}}}},
					},
					Jobs: []Job{
						{Name: "policy", Phase: PhasePostPlan},
						{Name: "summary", Phase: PhaseFinalize},
					},
				}
			default:
				t.Fatalf("unhandled test case %q", tt.name)
			}

			result := ir.DryRun(tt.totalModules)
			if result.Jobs != tt.wantJobs {
				t.Errorf("Jobs = %d, want %d", result.Jobs, tt.wantJobs)
			}
			if result.Stages != tt.wantStages {
				t.Errorf("Stages = %d, want %d", result.Stages, tt.wantStages)
			}
			if result.AffectedModules != tt.wantAffected {
				t.Errorf("AffectedModules = %d, want %d", result.AffectedModules, tt.wantAffected)
			}
			if result.TotalModules != tt.wantTotal {
				t.Errorf("TotalModules = %d, want %d", result.TotalModules, tt.wantTotal)
			}
			if result.ExecutionOrder == nil {
				t.Error("ExecutionOrder should not be nil")
			}
		})
	}
}
