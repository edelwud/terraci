package graph

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
)

func TestTopologicalSort(t *testing.T) {
	t.Parallel()

	g := buildTestGraph()

	sorted, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort: %v", err)
	}
	if len(sorted) != 4 {
		t.Fatalf("expected 4 modules, got %d", len(sorted))
	}
	if sorted[0] != "platform/stage/eu-central-1/vpc" {
		t.Errorf("first = %s, want vpc", sorted[0])
	}
	if sorted[len(sorted)-1] != "platform/stage/eu-central-1/app" {
		t.Errorf("last = %s, want app", sorted[len(sorted)-1])
	}
}

func TestTopologicalSort_Cycle(t *testing.T) {
	t.Parallel()

	modules := []*discovery.Module{
		discovery.TestModule("svc", "env", "reg", "a"),
		discovery.TestModule("svc", "env", "reg", "b"),
		discovery.TestModule("svc", "env", "reg", "c"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/a": {DependsOn: []string{"svc/env/reg/c"}},
		"svc/env/reg/b": {DependsOn: []string{"svc/env/reg/a"}},
		"svc/env/reg/c": {DependsOn: []string{"svc/env/reg/b"}},
	}
	g := BuildFromDependencies(modules, deps)

	if _, err := g.TopologicalSort(); err == nil {
		t.Error("expected cycle error")
	}
}

func TestExecutionLevels(t *testing.T) {
	t.Parallel()

	g := buildTestGraph()

	levels, err := g.ExecutionLevels()
	if err != nil {
		t.Fatalf("ExecutionLevels: %v", err)
	}
	if len(levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(levels))
	}
	if len(levels[0]) != 1 {
		t.Errorf("level 0: expected 1 module (vpc), got %d", len(levels[0]))
	}
	if len(levels[1]) != 2 {
		t.Errorf("level 1: expected 2 modules (eks, rds), got %d", len(levels[1]))
	}
	if len(levels[2]) != 1 {
		t.Errorf("level 2: expected 1 module (app), got %d", len(levels[2]))
	}
}

func TestDetectCycles(t *testing.T) {
	t.Parallel()

	modules := []*discovery.Module{
		discovery.TestModule("svc", "env", "reg", "a"),
		discovery.TestModule("svc", "env", "reg", "b"),
		discovery.TestModule("svc", "env", "reg", "c"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/a": {DependsOn: []string{"svc/env/reg/c"}},
		"svc/env/reg/b": {DependsOn: []string{"svc/env/reg/a"}},
		"svc/env/reg/c": {DependsOn: []string{"svc/env/reg/b"}},
	}
	g := BuildFromDependencies(modules, deps)

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Error("expected to detect cycles")
	}
}

func TestDetectCycles_NoCycles(t *testing.T) {
	t.Parallel()

	g := buildTestGraph()
	if cycles := g.DetectCycles(); len(cycles) != 0 {
		t.Errorf("expected no cycles, got %v", cycles)
	}
}
