package pipeline

import (
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/graph"
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
	modA := discovery.TestModule("svc", "prod", "us-east-1", "vpc")
	modB := discovery.TestModule("svc", "prod", "us-east-1", "rds")
	allModules := []*discovery.Module{modA, modB}
	idx := discovery.NewModuleIndex(allModules)
	depGraph := buildGraph(allModules, [][2]int{{1, 0}}) // B depends on A

	tests := []struct {
		name           string
		targets        []*discovery.Module
		isPR           bool
		isPolicy       bool
		isPlan         bool
		wantCount      int
		wantSummary    bool
		wantPolicy     bool
		wantLevelCount int
	}{
		{
			name:           "empty targets falls back to allModules",
			targets:        nil,
			isPR:           false,
			isPolicy:       false,
			isPlan:         false,
			wantCount:      2,
			wantSummary:    false,
			wantPolicy:     false,
			wantLevelCount: 2,
		},
		{
			name:           "non-empty targets used directly",
			targets:        []*discovery.Module{modA},
			isPR:           false,
			isPolicy:       false,
			isPlan:         false,
			wantCount:      1,
			wantSummary:    false,
			wantPolicy:     false,
			wantLevelCount: 1,
		},
		{
			name:        "PR and plan enabled sets IncludeSummary",
			targets:     allModules,
			isPR:        true,
			isPolicy:    false,
			isPlan:      true,
			wantCount:   2,
			wantSummary: true,
			wantPolicy:  false,
		},
		{
			name:        "policy and plan enabled sets IncludePolicy",
			targets:     allModules,
			isPR:        false,
			isPolicy:    true,
			isPlan:      true,
			wantCount:   2,
			wantSummary: false,
			wantPolicy:  true,
		},
		{
			name:        "PR enabled but plan disabled does not set IncludeSummary",
			targets:     allModules,
			isPR:        true,
			isPolicy:    false,
			isPlan:      false,
			wantCount:   2,
			wantSummary: false,
			wantPolicy:  false,
		},
		{
			name:        "policy enabled but plan disabled does not set IncludePolicy",
			targets:     allModules,
			isPR:        false,
			isPolicy:    true,
			isPlan:      false,
			wantCount:   2,
			wantSummary: false,
			wantPolicy:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := BuildJobPlan(depGraph, tt.targets, allModules, idx, tt.isPR, tt.isPolicy, tt.isPlan)
			if err != nil {
				t.Fatalf("BuildJobPlan() error = %v", err)
			}

			if got := len(plan.TargetModules); got != tt.wantCount {
				t.Errorf("TargetModules count = %d, want %d", got, tt.wantCount)
			}
			if plan.IncludeSummary != tt.wantSummary {
				t.Errorf("IncludeSummary = %v, want %v", plan.IncludeSummary, tt.wantSummary)
			}
			if plan.IncludePolicy != tt.wantPolicy {
				t.Errorf("IncludePolicy = %v, want %v", plan.IncludePolicy, tt.wantPolicy)
			}
			if tt.wantLevelCount > 0 && len(plan.ExecutionLevels) != tt.wantLevelCount {
				t.Errorf("ExecutionLevels count = %d, want %d", len(plan.ExecutionLevels), tt.wantLevelCount)
			}
			if plan.Subgraph == nil {
				t.Error("Subgraph should not be nil")
			}
			if plan.TargetSet == nil {
				t.Error("TargetSet should not be nil")
			}
			for _, m := range plan.TargetModules {
				if !plan.TargetSet[m.ID()] {
					t.Errorf("TargetSet missing module %s", m.ID())
				}
			}
		})
	}
}

func TestJobName(t *testing.T) {
	tests := []struct {
		name     string
		jobType  string
		module   *discovery.Module
		expected string
	}{
		{
			name:     "simple path",
			jobType:  "plan",
			module:   discovery.TestModule("svc", "prod", "us-east-1", "vpc"),
			expected: "plan-svc-prod-us-east-1-vpc",
		},
		{
			name:     "different job type",
			jobType:  "apply",
			module:   discovery.TestModule("svc", "prod", "us-east-1", "vpc"),
			expected: "apply-svc-prod-us-east-1-vpc",
		},
		{
			name:     "path with many segments",
			jobType:  "validate",
			module:   discovery.TestModule("payments", "staging", "eu-west-1", "rds"),
			expected: "validate-payments-staging-eu-west-1-rds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JobName(tt.jobType, tt.module)
			if got != tt.expected {
				t.Errorf("JobName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestResolveDependencyNames(t *testing.T) {
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
		jobType     string
		targetSet   map[string]bool
		wantNames   []string
		wantNoNames bool
	}{
		{
			name:    "dep in target set returns job name",
			module:  modB,
			jobType: "plan",
			targetSet: map[string]bool{
				modA.ID(): true,
				modB.ID(): true,
			},
			wantNames: []string{"plan-svc-prod-us-east-1-vpc"},
		},
		{
			name:    "dep not in target set is excluded",
			module:  modB,
			jobType: "plan",
			targetSet: map[string]bool{
				modB.ID(): true,
			},
			wantNoNames: true,
		},
		{
			name:    "module with no dependencies returns empty",
			module:  modA,
			jobType: "plan",
			targetSet: map[string]bool{
				modA.ID(): true,
				modB.ID(): true,
			},
			wantNoNames: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveDependencyNames(tt.module, tt.jobType, tt.targetSet, depGraph, idx)
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
	modA := discovery.TestModule("svc", "prod", "us-east-1", "vpc")
	modB := discovery.TestModule("svc", "prod", "us-east-1", "rds")

	// Build graph with both modules
	depGraph := buildGraph([]*discovery.Module{modA, modB}, [][2]int{{1, 0}})

	// Build index with only modB (modA is missing from index)
	idx := discovery.NewModuleIndex([]*discovery.Module{modB})

	targetSet := map[string]bool{
		modA.ID(): true,
		modB.ID(): true,
	}

	got := ResolveDependencyNames(modB, "plan", targetSet, depGraph, idx)
	// modA is in target set but not in index, so it should be skipped
	if len(got) != 0 {
		t.Errorf("expected no names when dep not in index, got %v", got)
	}
}

func TestBuildDryRunResult(t *testing.T) {
	modA := discovery.TestModule("svc", "prod", "us-east-1", "vpc")
	modB := discovery.TestModule("svc", "prod", "us-east-1", "rds")

	tests := []struct {
		name         string
		plan         *JobPlan
		totalModules int
		planEnabled  bool
		wantJobs     int
		wantStages   int
		wantAffected int
		wantTotal    int
	}{
		{
			name: "basic no policy no summary",
			plan: &JobPlan{
				TargetModules:   []*discovery.Module{modA, modB},
				ExecutionLevels: [][]string{{modA.ID()}, {modB.ID()}},
				IncludeSummary:  false,
				IncludePolicy:   false,
			},
			totalModules: 5,
			planEnabled:  false,
			wantJobs:     2,
			wantStages:   2,
			wantAffected: 2,
			wantTotal:    5,
		},
		{
			name: "planEnabled doubles job count per level",
			plan: &JobPlan{
				TargetModules:   []*discovery.Module{modA, modB},
				ExecutionLevels: [][]string{{modA.ID()}, {modB.ID()}},
				IncludeSummary:  false,
				IncludePolicy:   false,
			},
			totalModules: 5,
			planEnabled:  true,
			wantJobs:     4, // 2 levels * 1 module * 2 (plan+apply)
			wantStages:   2,
			wantAffected: 2,
			wantTotal:    5,
		},
		{
			name: "with policy adds 1 job and 1 stage",
			plan: &JobPlan{
				TargetModules:   []*discovery.Module{modA},
				ExecutionLevels: [][]string{{modA.ID()}},
				IncludeSummary:  false,
				IncludePolicy:   true,
			},
			totalModules: 3,
			planEnabled:  false,
			wantJobs:     2, // 1 module + 1 policy
			wantStages:   2, // 1 level + 1 policy
			wantAffected: 1,
			wantTotal:    3,
		},
		{
			name: "with summary adds 1 job and 1 stage",
			plan: &JobPlan{
				TargetModules:   []*discovery.Module{modA},
				ExecutionLevels: [][]string{{modA.ID()}},
				IncludeSummary:  true,
				IncludePolicy:   false,
			},
			totalModules: 3,
			planEnabled:  false,
			wantJobs:     2, // 1 module + 1 summary
			wantStages:   2, // 1 level + 1 summary
			wantAffected: 1,
			wantTotal:    3,
		},
		{
			name: "with both policy and summary",
			plan: &JobPlan{
				TargetModules:   []*discovery.Module{modA, modB},
				ExecutionLevels: [][]string{{modA.ID(), modB.ID()}},
				IncludeSummary:  true,
				IncludePolicy:   true,
			},
			totalModules: 10,
			planEnabled:  false,
			wantJobs:     4, // 2 modules + 1 policy + 1 summary
			wantStages:   3, // 1 level + 1 policy + 1 summary
			wantAffected: 2,
			wantTotal:    10,
		},
		{
			name: "planEnabled with policy and summary",
			plan: &JobPlan{
				TargetModules:   []*discovery.Module{modA, modB},
				ExecutionLevels: [][]string{{modA.ID(), modB.ID()}},
				IncludeSummary:  true,
				IncludePolicy:   true,
			},
			totalModules: 10,
			planEnabled:  true,
			wantJobs:     6, // 2*2 modules (plan+apply) + 1 policy + 1 summary
			wantStages:   3,
			wantAffected: 2,
			wantTotal:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildDryRunResult(tt.plan, tt.totalModules, tt.planEnabled)
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
