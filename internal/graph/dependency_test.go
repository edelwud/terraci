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
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
		discovery.TestModule("platform", "stage", "eu-central-1", "rds"),
		discovery.TestModule("platform", "stage", "eu-central-1", "app"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/rds": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/app": {DependsOn: []string{"platform/stage/eu-central-1/eks", "platform/stage/eu-central-1/rds"}},
	}

	g := BuildFromDependencies(modules, deps)

	sorted, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	// vpc should come first
	if sorted[0] != "platform/stage/eu-central-1/vpc" {
		t.Errorf("Expected vpc first, got %s", sorted[0])
	}

	// app should come last
	if sorted[len(sorted)-1] != "platform/stage/eu-central-1/app" {
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
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
		discovery.TestModule("platform", "stage", "eu-central-1", "rds"),
		discovery.TestModule("platform", "stage", "eu-central-1", "app"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/rds": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/app": {DependsOn: []string{"platform/stage/eu-central-1/eks", "platform/stage/eu-central-1/rds"}},
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
	if len(levels[0]) != 1 || levels[0][0] != "platform/stage/eu-central-1/vpc" {
		t.Errorf("Level 0 should contain only vpc, got %v", levels[0])
	}

	// Level 1: eks, rds (order doesn't matter)
	if len(levels[1]) != 2 {
		t.Errorf("Level 1 should contain 2 modules, got %d", len(levels[1]))
	}

	// Level 2: app
	if len(levels[2]) != 1 || levels[2][0] != "platform/stage/eu-central-1/app" {
		t.Errorf("Level 2 should contain only app, got %v", levels[2])
	}
}

func TestDependencyGraph_GetAffectedModules(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
		discovery.TestModule("platform", "stage", "eu-central-1", "rds"),
		discovery.TestModule("platform", "stage", "eu-central-1", "app"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/rds": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/app": {DependsOn: []string{"platform/stage/eu-central-1/eks", "platform/stage/eu-central-1/rds"}},
	}

	g := BuildFromDependencies(modules, deps)

	// If vpc changes, all modules are affected (vpc + all dependents)
	affected := g.GetAffectedModules([]string{"platform/stage/eu-central-1/vpc"})
	sort.Strings(affected)

	expected := []string{
		"platform/stage/eu-central-1/app",
		"platform/stage/eu-central-1/eks",
		"platform/stage/eu-central-1/rds",
		"platform/stage/eu-central-1/vpc",
	}

	if !reflect.DeepEqual(affected, expected) {
		t.Errorf("Expected %v, got %v", expected, affected)
	}

	// If eks changes: eks + dependents (app) + dependencies (vpc)
	affected = g.GetAffectedModules([]string{"platform/stage/eu-central-1/eks"})
	sort.Strings(affected)

	expected = []string{
		"platform/stage/eu-central-1/app",
		"platform/stage/eu-central-1/eks",
		"platform/stage/eu-central-1/vpc", // dependency of eks
	}

	if !reflect.DeepEqual(affected, expected) {
		t.Errorf("Expected %v, got %v", expected, affected)
	}

	// If app changes: app + dependencies (eks, rds) + transitive dependencies (vpc)
	affected = g.GetAffectedModules([]string{"platform/stage/eu-central-1/app"})
	sort.Strings(affected)

	expected = []string{
		"platform/stage/eu-central-1/app",
		"platform/stage/eu-central-1/eks", // dependency of app
		"platform/stage/eu-central-1/rds", // dependency of app
		"platform/stage/eu-central-1/vpc", // transitive dependency (eks->vpc, rds->vpc)
	}

	if !reflect.DeepEqual(affected, expected) {
		t.Errorf("Expected %v, got %v", expected, affected)
	}
}

func TestDependencyGraph_Subgraph(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
		discovery.TestModule("platform", "stage", "eu-central-1", "rds"),
		discovery.TestModule("platform", "stage", "eu-central-1", "app"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/rds": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/app": {DependsOn: []string{"platform/stage/eu-central-1/eks", "platform/stage/eu-central-1/rds"}},
	}

	g := BuildFromDependencies(modules, deps)

	// Create subgraph with only vpc, eks, app
	sub := g.Subgraph([]string{
		"platform/stage/eu-central-1/vpc",
		"platform/stage/eu-central-1/eks",
		"platform/stage/eu-central-1/app",
	})

	if len(sub.Nodes()) != 3 {
		t.Errorf("Expected 3 nodes in subgraph, got %d", len(sub.Nodes()))
	}

	// app should only depend on eks in subgraph (rds is excluded)
	appDeps := sub.GetDependencies("platform/stage/eu-central-1/app")
	if len(appDeps) != 1 || appDeps[0] != "platform/stage/eu-central-1/eks" {
		t.Errorf("Expected app to depend only on eks, got %v", appDeps)
	}
}

func TestDependencyGraph_ToDOT(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
	}

	g := BuildFromDependencies(modules, deps)

	dot := g.ToDOT()

	// Basic sanity checks
	if !contains(dot, "digraph dependencies") {
		t.Error("DOT output missing digraph declaration")
	}

	if !contains(dot, "platform/stage/eu-central-1/vpc") {
		t.Error("DOT output missing vpc node")
	}

	if !contains(dot, "->") {
		t.Error("DOT output missing edge")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDependencyGraph_LibraryModuleTracking(t *testing.T) {
	// Create graph with library module usage
	// msk module uses library modules: _modules/kafka, _modules/kafka_acl
	// kafka_acl depends on kafka (both are library modules)
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-north-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-north-1", "msk"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-north-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-north-1/msk": {
			DependsOn: []string{"platform/stage/eu-north-1/vpc"},
			LibraryDependencies: []*parser.LibraryDependency{
				{LibraryPath: "/project/_modules/kafka"},
				{LibraryPath: "/project/_modules/kafka_acl"},
			},
		},
	}

	g := BuildFromDependencies(modules, deps)

	// Check library usage is tracked
	kafkaUsers := g.GetModulesUsingLibrary("/project/_modules/kafka")
	if len(kafkaUsers) != 1 || kafkaUsers[0] != "platform/stage/eu-north-1/msk" {
		t.Errorf("Expected msk to use kafka, got %v", kafkaUsers)
	}

	// Check all library paths
	allPaths := g.GetAllLibraryPaths()
	if len(allPaths) != 2 {
		t.Errorf("Expected 2 library paths, got %d: %v", len(allPaths), allPaths)
	}
}

func TestDependencyGraph_GetAffectedByLibraryChanges(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-north-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-north-1", "msk"),
		discovery.TestModule("platform", "stage", "eu-north-1", "app"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-north-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-north-1/msk": {
			DependsOn: []string{"platform/stage/eu-north-1/vpc"},
			LibraryDependencies: []*parser.LibraryDependency{
				{LibraryPath: "/project/_modules/kafka"},
			},
		},
		"platform/stage/eu-north-1/app": {
			DependsOn: []string{"platform/stage/eu-north-1/msk"},
		},
	}

	g := BuildFromDependencies(modules, deps)

	// When kafka library changes, msk should be affected
	affected := g.GetAffectedByLibraryChanges([]string{"/project/_modules/kafka"})
	if len(affected) != 1 || affected[0] != "platform/stage/eu-north-1/msk" {
		t.Errorf("Expected only msk to be affected, got %v", affected)
	}
}

func TestDependencyGraph_GetAffectedModulesWithLibraries(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-north-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-north-1", "msk"),
		discovery.TestModule("platform", "stage", "eu-north-1", "app"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-north-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-north-1/msk": {
			DependsOn: []string{"platform/stage/eu-north-1/vpc"},
			LibraryDependencies: []*parser.LibraryDependency{
				{LibraryPath: "/project/_modules/kafka"},
			},
		},
		"platform/stage/eu-north-1/app": {
			DependsOn: []string{"platform/stage/eu-north-1/msk"},
		},
	}

	g := BuildFromDependencies(modules, deps)

	// When kafka library changes, msk and its dependencies (vpc) should be affected
	affected := g.GetAffectedModulesWithLibraries([]string{}, []string{"/project/_modules/kafka"})
	sort.Strings(affected)

	expected := []string{
		"platform/stage/eu-north-1/msk",
		"platform/stage/eu-north-1/vpc", // dependency of msk
	}

	if !reflect.DeepEqual(affected, expected) {
		t.Errorf("Expected %v, got %v", expected, affected)
	}

	// When both an executable module and a library change
	affected = g.GetAffectedModulesWithLibraries(
		[]string{"platform/stage/eu-north-1/app"},
		[]string{"/project/_modules/kafka"},
	)
	sort.Strings(affected)

	expected = []string{
		"platform/stage/eu-north-1/app",
		"platform/stage/eu-north-1/msk",
		"platform/stage/eu-north-1/vpc",
	}

	if !reflect.DeepEqual(affected, expected) {
		t.Errorf("Expected %v, got %v", expected, affected)
	}
}

func TestDependencyGraph_GetStats(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
		discovery.TestModule("platform", "stage", "eu-central-1", "rds"),
		discovery.TestModule("platform", "stage", "eu-central-1", "app"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/rds": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
		"platform/stage/eu-central-1/app": {DependsOn: []string{"platform/stage/eu-central-1/eks", "platform/stage/eu-central-1/rds"}},
	}

	g := BuildFromDependencies(modules, deps)
	stats := g.GetStats()

	if stats.TotalModules != 4 {
		t.Errorf("TotalModules = %d, want 4", stats.TotalModules)
	}
	if stats.TotalEdges != 4 {
		t.Errorf("TotalEdges = %d, want 4", stats.TotalEdges)
	}
	if stats.RootModules != 1 {
		t.Errorf("RootModules = %d, want 1 (vpc)", stats.RootModules)
	}
	if stats.LeafModules != 1 {
		t.Errorf("LeafModules = %d, want 1 (app)", stats.LeafModules)
	}
	if stats.MaxDepth != 2 {
		t.Errorf("MaxDepth = %d, want 2", stats.MaxDepth)
	}
	if stats.HasCycles {
		t.Error("HasCycles should be false")
	}
	if stats.CycleCount != 0 {
		t.Errorf("CycleCount = %d, want 0", stats.CycleCount)
	}
}

func TestDependencyGraph_GetStats_Empty(t *testing.T) {
	g := NewDependencyGraph()
	stats := g.GetStats()

	if stats.TotalModules != 0 {
		t.Errorf("TotalModules = %d, want 0", stats.TotalModules)
	}
	if stats.TotalEdges != 0 {
		t.Errorf("TotalEdges = %d, want 0", stats.TotalEdges)
	}
}

func TestDependencyGraph_GetNode(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "env", "reg", "vpc"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/vpc": {DependsOn: []string{}},
	}
	g := BuildFromDependencies(modules, deps)

	node := g.GetNode("svc/env/reg/vpc")
	if node == nil {
		t.Fatal("GetNode returned nil for existing node")
	}
	if node.Module.Get("module") != "vpc" {
		t.Errorf("GetNode module = %q, want %q", node.Module.Get("module"), "vpc")
	}

	nonExistent := g.GetNode("nonexistent")
	if nonExistent != nil {
		t.Error("GetNode should return nil for nonexistent node")
	}
}

func TestDependencyGraph_GetDependencies_GetDependents(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "env", "reg", "a"),
		discovery.TestModule("svc", "env", "reg", "b"),
		discovery.TestModule("svc", "env", "reg", "c"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/a": {DependsOn: []string{}},
		"svc/env/reg/b": {DependsOn: []string{"svc/env/reg/a"}},
		"svc/env/reg/c": {DependsOn: []string{"svc/env/reg/a"}},
	}
	g := BuildFromDependencies(modules, deps)

	// b depends on a
	bDeps := g.GetDependencies("svc/env/reg/b")
	if len(bDeps) != 1 || bDeps[0] != "svc/env/reg/a" {
		t.Errorf("GetDependencies(b) = %v, want [svc/env/reg/a]", bDeps)
	}

	// a has no dependencies
	aDeps := g.GetDependencies("svc/env/reg/a")
	if len(aDeps) != 0 {
		t.Errorf("GetDependencies(a) = %v, want empty", aDeps)
	}

	// a is depended on by b and c
	aDependents := g.GetDependents("svc/env/reg/a")
	if len(aDependents) != 2 {
		t.Errorf("GetDependents(a) length = %d, want 2", len(aDependents))
	}

	// nonexistent module returns nil
	nDeps := g.GetDependencies("nonexistent")
	if nDeps != nil {
		t.Errorf("GetDependencies(nonexistent) = %v, want nil", nDeps)
	}
}

func TestDependencyGraph_AddEdge_Dedup(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "env", "reg", "a"),
		discovery.TestModule("svc", "env", "reg", "b"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/a": {DependsOn: []string{}},
		"svc/env/reg/b": {DependsOn: []string{"svc/env/reg/a"}},
	}
	g := BuildFromDependencies(modules, deps)

	// Add duplicate edge
	g.AddEdge("svc/env/reg/b", "svc/env/reg/a")

	bDeps := g.GetDependencies("svc/env/reg/b")
	if len(bDeps) != 1 {
		t.Errorf("duplicate AddEdge should not create duplicate, got %d edges", len(bDeps))
	}
}

func TestDependencyGraph_AddEdge_NonexistentNodes(t *testing.T) {
	g := NewDependencyGraph()
	// Adding edge between nonexistent nodes should be a no-op
	g.AddEdge("from", "to")
	if len(g.Nodes()) != 0 {
		t.Error("AddEdge with nonexistent nodes should not create nodes")
	}
}

func TestDependencyGraph_ToPlantUML(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-central-1", "eks"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-central-1/vpc": {DependsOn: []string{}},
		"platform/stage/eu-central-1/eks": {DependsOn: []string{"platform/stage/eu-central-1/vpc"}},
	}
	g := BuildFromDependencies(modules, deps)

	plantuml := g.ToPlantUML()

	if !contains(plantuml, "@startuml") {
		t.Error("PlantUML output missing @startuml")
	}
	if !contains(plantuml, "@enduml") {
		t.Error("PlantUML output missing @enduml")
	}
	if !contains(plantuml, "platform/stage") {
		t.Error("PlantUML output missing package grouping")
	}
	if !contains(plantuml, "-->") {
		t.Error("PlantUML output missing edge")
	}
}

func TestPlantUMLAlias(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"platform/stage/eu-central-1/vpc", "platform_stage_eu_central_1_vpc"},
		{"simple", "simple"},
		{"with.dots/and-dashes", "with_dots_and_dashes"},
	}

	for _, tt := range tests {
		got := plantUMLAlias(tt.input)
		if got != tt.want {
			t.Errorf("plantUMLAlias(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeLibraryPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/project/_modules/kafka/", "/project/_modules/kafka"},
		{"/project/_modules/kafka", "/project/_modules/kafka"},
		{"relative/path/", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeLibraryPath(tt.input)
		if got != tt.want {
			t.Errorf("normalizeLibraryPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDependencyGraph_AddLibraryUsage_Dedup(t *testing.T) {
	g := NewDependencyGraph()
	g.AddLibraryUsage("/lib/kafka", "module-a")
	g.AddLibraryUsage("/lib/kafka", "module-a") // duplicate

	users := g.GetModulesUsingLibrary("/lib/kafka")
	if len(users) != 1 {
		t.Errorf("expected 1 user after dedup, got %d", len(users))
	}
}

func TestDependencyGraph_GetAllDependencies(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "env", "reg", "a"),
		discovery.TestModule("svc", "env", "reg", "b"),
		discovery.TestModule("svc", "env", "reg", "c"),
	}
	// c depends on b depends on a
	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/a": {DependsOn: []string{}},
		"svc/env/reg/b": {DependsOn: []string{"svc/env/reg/a"}},
		"svc/env/reg/c": {DependsOn: []string{"svc/env/reg/b"}},
	}
	g := BuildFromDependencies(modules, deps)

	allDeps := g.GetAllDependencies("svc/env/reg/c")
	sort.Strings(allDeps)
	expected := []string{"svc/env/reg/a", "svc/env/reg/b"}
	if !reflect.DeepEqual(allDeps, expected) {
		t.Errorf("GetAllDependencies(c) = %v, want %v", allDeps, expected)
	}
}

func TestDependencyGraph_GetAllDependents(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "env", "reg", "a"),
		discovery.TestModule("svc", "env", "reg", "b"),
		discovery.TestModule("svc", "env", "reg", "c"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/a": {DependsOn: []string{}},
		"svc/env/reg/b": {DependsOn: []string{"svc/env/reg/a"}},
		"svc/env/reg/c": {DependsOn: []string{"svc/env/reg/b"}},
	}
	g := BuildFromDependencies(modules, deps)

	allDependents := g.GetAllDependents("svc/env/reg/a")
	sort.Strings(allDependents)
	expected := []string{"svc/env/reg/b", "svc/env/reg/c"}
	if !reflect.DeepEqual(allDependents, expected) {
		t.Errorf("GetAllDependents(a) = %v, want %v", allDependents, expected)
	}
}

func TestDependencyGraph_TransitiveLibraryDependencies(t *testing.T) {
	// Test case: kafka_acl library is under kafka directory
	// Changing kafka should affect modules using kafka_acl too
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-north-1", "msk"),
	}

	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-north-1/msk": {
			LibraryDependencies: []*parser.LibraryDependency{
				{LibraryPath: "/project/_modules/kafka/acl"},
			},
		},
	}

	g := BuildFromDependencies(modules, deps)

	// When parent directory kafka changes, modules using kafka/acl should be affected
	affected := g.GetAffectedByLibraryChanges([]string{"/project/_modules/kafka"})
	if len(affected) != 1 || affected[0] != "platform/stage/eu-north-1/msk" {
		t.Errorf("Expected msk to be affected by parent library change, got %v", affected)
	}
}
