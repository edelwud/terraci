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

func TestDependencyNames(t *testing.T) {
	t.Parallel()

	deps := []JobDependency{
		{Job: "plan-a"},
		{Job: ""},
		{Job: "plan-b"},
	}
	got := DependencyNames(deps)
	want := []string{"plan-a", "plan-b"}
	if len(got) != len(want) {
		t.Fatalf("DependencyNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DependencyNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPrepareModuleGraph(t *testing.T) {
	t.Parallel()

	modA := discovery.TestModule("svc", "prod", "us-east-1", "vpc")
	modB := discovery.TestModule("svc", "prod", "us-east-1", "rds")
	allModules := []*discovery.Module{modA, modB}
	idx := discovery.NewModuleIndex(allModules)
	depGraph := buildGraph(allModules, [][2]int{{1, 0}}) // B depends on A

	tests := []struct {
		name          string
		targets       []*discovery.Module
		wantCount     int
		wantOrderSize int
	}{
		{
			name:          "empty targets falls back to allModules",
			targets:       nil,
			wantCount:     2,
			wantOrderSize: 2,
		},
		{
			name:          "non-empty targets used directly",
			targets:       []*discovery.Module{modA},
			wantCount:     1,
			wantOrderSize: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan, err := prepareModuleGraph(depGraph, tt.targets, allModules, idx)
			if err != nil {
				t.Fatalf("prepareModuleGraph() error = %v", err)
			}

			if got := len(plan.targetModules); got != tt.wantCount {
				t.Errorf("target module count = %d, want %d", got, tt.wantCount)
			}
			if got := len(plan.moduleOrder); got != tt.wantOrderSize {
				t.Errorf("module order count = %d, want %d", got, tt.wantOrderSize)
			}
			if plan.subgraph == nil {
				t.Error("subgraph should not be nil")
			}
			for _, m := range plan.targetModules {
				if plan.subgraph.GetNode(m.ID()) == nil {
					t.Errorf("subgraph missing target module %s", m.ID())
				}
			}
		})
	}
}

func TestJobNameInternal(t *testing.T) {
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
			name:     "command kind has no prefix",
			jobKind:  JobKindCommand,
			module:   discovery.TestModule("payments", "staging", "eu-west-1", "rds"),
			expected: "payments-staging-eu-west-1-rds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := jobName(tt.jobKind, tt.module)
			if got != tt.expected {
				t.Errorf("jobName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestResolveDependencyNamesInternal(t *testing.T) {
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
			got := resolveDependencyNames(tt.module, tt.jobKind, subgraph, idx)
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

func TestResolveDependencyNamesInternal_DepNotInIndex(t *testing.T) {
	t.Parallel()

	modA := discovery.TestModule("svc", "prod", "us-east-1", "vpc")
	modB := discovery.TestModule("svc", "prod", "us-east-1", "rds")

	// Build graph with both modules
	depGraph := buildGraph([]*discovery.Module{modA, modB}, [][2]int{{1, 0}})

	// Build index with only modB (modA is missing from index)
	idx := discovery.NewModuleIndex([]*discovery.Module{modB})

	subgraph := depGraph.Subgraph([]string{modA.ID(), modB.ID()})
	got := resolveDependencyNames(modB, JobKindPlan, subgraph, idx)
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
			wantStages:   4,
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
			name:         "independent contributed jobs share one DAG layer",
			totalModules: 10,
			wantJobs:     6, // 2*2 module jobs + 2 contributed jobs
			wantStages:   1,
			wantAffected: 2,
			wantTotal:    10,
		},
		{
			name:         "contributed dependencies increase stage count",
			totalModules: 2,
			wantJobs:     3,
			wantStages:   2,
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
					jobs: []Job{
						{name: "apply-a", module: modA},
						{name: "apply-b", module: modB, dependencies: []JobDependency{{Job: "apply-a"}}},
					},
				}
			case "planEnabled doubles job count per level":
				ir = &IR{
					jobs: []Job{
						{name: "plan-a", module: modA},
						{name: "apply-a", module: modA, dependencies: []JobDependency{{Job: "plan-a"}}},
						{name: "plan-b", module: modB, dependencies: []JobDependency{{Job: "apply-a"}}},
						{name: "apply-b", module: modB, dependencies: []JobDependency{{Job: "plan-b"}}},
					},
				}
			case "contributed jobs add 1 job and 1 stage":
				ir = &IR{
					jobs: []Job{
						{name: "apply-a", module: modA},
						{name: "summary", dependencies: []JobDependency{{Job: "apply-a"}}},
					},
				}
			case "independent contributed jobs share one DAG layer":
				ir = &IR{
					jobs: []Job{
						{name: "plan-a", module: modA},
						{name: "apply-a", module: modA},
						{name: "plan-b", module: modB},
						{name: "apply-b", module: modB},
						{name: "policy"},
						{name: "cost"},
					},
				}
			case "contributed dependencies increase stage count":
				ir = &IR{
					jobs: []Job{
						{name: "apply-a", module: modA},
						{name: "policy"},
						{name: "summary", dependencies: []JobDependency{{Job: "policy"}}},
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
			if result.JobGroups == nil {
				t.Error("JobGroups should not be nil")
			}
		})
	}
}
