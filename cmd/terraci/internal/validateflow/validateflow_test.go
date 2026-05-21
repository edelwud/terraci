package validateflow

import (
	"testing"

	"github.com/edelwud/terraci/cmd/terraci/internal/projectflow"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/pkg/workflow"
)

func TestEvaluatePassesAcyclicGraph(t *testing.T) {
	vpc := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	eks := discovery.TestModule("platform", "stage", "eu-central-1", "eks")
	depGraph := graph.BuildFromDependencies([]*discovery.Module{vpc, eks}, map[string]*parser.ModuleDependencies{
		eks.ID(): {DependsOn: []string{vpc.ID()}},
	})

	result := Evaluate(&projectflow.Result{Workflow: &workflow.Result{
		Graph: depGraph,
		Dependencies: map[string]*parser.ModuleDependencies{
			eks.ID(): {DependsOn: []string{vpc.ID()}},
		},
	}})

	if !result.Passed {
		t.Fatal("Passed = false, want true")
	}
	if result.DependencyLinks != 1 {
		t.Fatalf("DependencyLinks = %d, want 1", result.DependencyLinks)
	}
	if len(result.ExecutionLevels) == 0 {
		t.Fatal("ExecutionLevels is empty")
	}
}

func TestEvaluateFailsCycles(t *testing.T) {
	first := discovery.TestModule("platform", "stage", "eu-central-1", "first")
	second := discovery.TestModule("platform", "stage", "eu-central-1", "second")
	depGraph := graph.NewDependencyGraph()
	depGraph.AddNode(first)
	depGraph.AddNode(second)
	depGraph.AddEdge(first.ID(), second.ID())
	depGraph.AddEdge(second.ID(), first.ID())

	result := Evaluate(&projectflow.Result{Workflow: &workflow.Result{Graph: depGraph}})

	if result.Passed {
		t.Fatal("Passed = true, want false")
	}
	if len(result.Cycles) == 0 {
		t.Fatal("Cycles is empty")
	}
	if result.ExecutionLevelsError == nil {
		t.Fatal("ExecutionLevelsError = nil")
	}
}
