package graph

import (
	"reflect"
	"sort"
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/parser"
)

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	// Create a simple graph:
	// vpc -> eks -> app
	//     -> rds -> app

	modules := []*discovery.Module{
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "eks"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "rds"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "app"},
	}

	deps := map[string]*parser.ModuleDependencies{
		"cdp/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"cdp/stage/eu-central-1/eks": {DependsOn: []string{"cdp/stage/eu-central-1/vpc"}},
		"cdp/stage/eu-central-1/rds": {DependsOn: []string{"cdp/stage/eu-central-1/vpc"}},
		"cdp/stage/eu-central-1/app": {DependsOn: []string{"cdp/stage/eu-central-1/eks", "cdp/stage/eu-central-1/rds"}},
	}

	g := BuildFromDependencies(modules, deps)

	sorted, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// vpc should come first
	if sorted[0] != "cdp/stage/eu-central-1/vpc" {
		t.Errorf("Expected vpc first, got %s", sorted[0])
	}

	// app should come last
	if sorted[len(sorted)-1] != "cdp/stage/eu-central-1/app" {
		t.Errorf("Expected app last, got %s", sorted[len(sorted)-1])
	}

	// Verify all modules are present
	if len(sorted) != 4 {
		t.Errorf("Expected 4 modules, got %d", len(sorted))
	}
}

func TestDependencyGraph_CycleDetection(t *testing.T) {
	// Create a graph with a cycle:
	// a -> b -> c -> a

	modules := []*discovery.Module{
		{Service: "svc", Environment: "env", Region: "reg", Module: "a"},
		{Service: "svc", Environment: "env", Region: "reg", Module: "b"},
		{Service: "svc", Environment: "env", Region: "reg", Module: "c"},
	}

	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/a": {DependsOn: []string{"svc/env/reg/c"}},
		"svc/env/reg/b": {DependsOn: []string{"svc/env/reg/a"}},
		"svc/env/reg/c": {DependsOn: []string{"svc/env/reg/b"}},
	}

	g := BuildFromDependencies(modules, deps)

	_, err := g.TopologicalSort()
	if err == nil {
		t.Error("Expected cycle detection error, got nil")
	}

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Error("Expected to detect cycles, found none")
	}
}

func TestDependencyGraph_ExecutionLevels(t *testing.T) {
	// Create graph:
	// Level 0: vpc
	// Level 1: eks, rds (parallel)
	// Level 2: app

	modules := []*discovery.Module{
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "eks"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "rds"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "app"},
	}

	deps := map[string]*parser.ModuleDependencies{
		"cdp/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"cdp/stage/eu-central-1/eks": {DependsOn: []string{"cdp/stage/eu-central-1/vpc"}},
		"cdp/stage/eu-central-1/rds": {DependsOn: []string{"cdp/stage/eu-central-1/vpc"}},
		"cdp/stage/eu-central-1/app": {DependsOn: []string{"cdp/stage/eu-central-1/eks", "cdp/stage/eu-central-1/rds"}},
	}

	g := BuildFromDependencies(modules, deps)

	levels, err := g.ExecutionLevels()
	if err != nil {
		t.Fatalf("ExecutionLevels failed: %v", err)
	}

	if len(levels) != 3 {
		t.Errorf("Expected 3 levels, got %d", len(levels))
	}

	// Level 0: vpc
	if len(levels[0]) != 1 || levels[0][0] != "cdp/stage/eu-central-1/vpc" {
		t.Errorf("Level 0 should contain only vpc, got %v", levels[0])
	}

	// Level 1: eks, rds (order doesn't matter)
	if len(levels[1]) != 2 {
		t.Errorf("Level 1 should contain 2 modules, got %d", len(levels[1]))
	}

	// Level 2: app
	if len(levels[2]) != 1 || levels[2][0] != "cdp/stage/eu-central-1/app" {
		t.Errorf("Level 2 should contain only app, got %v", levels[2])
	}
}

func TestDependencyGraph_GetAffectedModules(t *testing.T) {
	modules := []*discovery.Module{
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "eks"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "rds"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "app"},
	}

	deps := map[string]*parser.ModuleDependencies{
		"cdp/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"cdp/stage/eu-central-1/eks": {DependsOn: []string{"cdp/stage/eu-central-1/vpc"}},
		"cdp/stage/eu-central-1/rds": {DependsOn: []string{"cdp/stage/eu-central-1/vpc"}},
		"cdp/stage/eu-central-1/app": {DependsOn: []string{"cdp/stage/eu-central-1/eks", "cdp/stage/eu-central-1/rds"}},
	}

	g := BuildFromDependencies(modules, deps)

	// If vpc changes, all modules are affected
	affected := g.GetAffectedModules([]string{"cdp/stage/eu-central-1/vpc"})
	sort.Strings(affected)

	expected := []string{
		"cdp/stage/eu-central-1/app",
		"cdp/stage/eu-central-1/eks",
		"cdp/stage/eu-central-1/rds",
		"cdp/stage/eu-central-1/vpc",
	}

	if !reflect.DeepEqual(affected, expected) {
		t.Errorf("Expected %v, got %v", expected, affected)
	}

	// If eks changes, only eks and app are affected
	affected = g.GetAffectedModules([]string{"cdp/stage/eu-central-1/eks"})
	sort.Strings(affected)

	expected = []string{
		"cdp/stage/eu-central-1/app",
		"cdp/stage/eu-central-1/eks",
	}

	if !reflect.DeepEqual(affected, expected) {
		t.Errorf("Expected %v, got %v", expected, affected)
	}
}

func TestDependencyGraph_Subgraph(t *testing.T) {
	modules := []*discovery.Module{
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "eks"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "rds"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "app"},
	}

	deps := map[string]*parser.ModuleDependencies{
		"cdp/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"cdp/stage/eu-central-1/eks": {DependsOn: []string{"cdp/stage/eu-central-1/vpc"}},
		"cdp/stage/eu-central-1/rds": {DependsOn: []string{"cdp/stage/eu-central-1/vpc"}},
		"cdp/stage/eu-central-1/app": {DependsOn: []string{"cdp/stage/eu-central-1/eks", "cdp/stage/eu-central-1/rds"}},
	}

	g := BuildFromDependencies(modules, deps)

	// Create subgraph with only vpc, eks, app
	sub := g.Subgraph([]string{
		"cdp/stage/eu-central-1/vpc",
		"cdp/stage/eu-central-1/eks",
		"cdp/stage/eu-central-1/app",
	})

	if len(sub.Nodes()) != 3 {
		t.Errorf("Expected 3 nodes in subgraph, got %d", len(sub.Nodes()))
	}

	// app should only depend on eks in subgraph (rds is excluded)
	appDeps := sub.GetDependencies("cdp/stage/eu-central-1/app")
	if len(appDeps) != 1 || appDeps[0] != "cdp/stage/eu-central-1/eks" {
		t.Errorf("Expected app to depend only on eks, got %v", appDeps)
	}
}

func TestDependencyGraph_ToDOT(t *testing.T) {
	modules := []*discovery.Module{
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc"},
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "eks"},
	}

	deps := map[string]*parser.ModuleDependencies{
		"cdp/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"cdp/stage/eu-central-1/eks": {DependsOn: []string{"cdp/stage/eu-central-1/vpc"}},
	}

	g := BuildFromDependencies(modules, deps)

	dot := g.ToDOT()

	// Basic sanity checks
	if !contains(dot, "digraph dependencies") {
		t.Error("DOT output missing digraph declaration")
	}

	if !contains(dot, "cdp/stage/eu-central-1/vpc") {
		t.Error("DOT output missing vpc node")
	}

	if !contains(dot, "->") {
		t.Error("DOT output missing edge")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
