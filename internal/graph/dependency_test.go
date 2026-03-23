package graph

import (
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
	"github.com/edelwud/terraci/internal/parser"
)

// --- Helpers ---

func buildTestGraph() *DependencyGraph {
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
	return BuildFromDependencies(modules, deps)
}

// --- Graph construction ---

func TestBuildFromDependencies(t *testing.T) {
	g := buildTestGraph()
	if len(g.Nodes()) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(g.Nodes()))
	}
}

func TestAddNode(t *testing.T) {
	g := NewDependencyGraph()
	m := discovery.TestModule("svc", "env", "reg", "vpc")
	g.AddNode(m)
	g.AddNode(m) // duplicate

	if len(g.Nodes()) != 1 {
		t.Errorf("duplicate AddNode should not create duplicate, got %d", len(g.Nodes()))
	}
}

func TestAddEdge(t *testing.T) {
	g := NewDependencyGraph()
	a := discovery.TestModule("svc", "env", "reg", "a")
	b := discovery.TestModule("svc", "env", "reg", "b")
	g.AddNode(a)
	g.AddNode(b)

	g.AddEdge(a.ID(), b.ID())
	g.AddEdge(a.ID(), b.ID()) // duplicate

	if deps := g.GetDependencies(a.ID()); len(deps) != 1 {
		t.Errorf("duplicate AddEdge should not create duplicate, got %d", len(deps))
	}
}

func TestAddEdge_NonexistentNodes(t *testing.T) {
	g := NewDependencyGraph()
	g.AddEdge("from", "to")
	if len(g.Nodes()) != 0 {
		t.Error("AddEdge with nonexistent nodes should not create nodes")
	}
}

func TestGetNode(t *testing.T) {
	g := buildTestGraph()

	node := g.GetNode("platform/stage/eu-central-1/vpc")
	if node == nil {
		t.Fatal("GetNode returned nil for existing node")
	}
	if node.Module.Get("module") != "vpc" {
		t.Errorf("module = %q, want vpc", node.Module.Get("module"))
	}

	if g.GetNode("nonexistent") != nil {
		t.Error("GetNode should return nil for nonexistent node")
	}
}

func TestGetDependencies_GetDependents(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "env", "reg", "a"),
		discovery.TestModule("svc", "env", "reg", "b"),
		discovery.TestModule("svc", "env", "reg", "c"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/a": {},
		"svc/env/reg/b": {DependsOn: []string{"svc/env/reg/a"}},
		"svc/env/reg/c": {DependsOn: []string{"svc/env/reg/a"}},
	}
	g := BuildFromDependencies(modules, deps)

	if d := g.GetDependencies("svc/env/reg/b"); len(d) != 1 || d[0] != "svc/env/reg/a" {
		t.Errorf("GetDependencies(b) = %v, want [svc/env/reg/a]", d)
	}
	if d := g.GetDependencies("svc/env/reg/a"); len(d) != 0 {
		t.Errorf("GetDependencies(a) = %v, want empty", d)
	}
	if d := g.GetDependents("svc/env/reg/a"); len(d) != 2 {
		t.Errorf("GetDependents(a) = %d, want 2", len(d))
	}
	if d := g.GetDependencies("nonexistent"); d != nil {
		t.Errorf("GetDependencies(nonexistent) = %v, want nil", d)
	}
}

func TestGetAllDependencies(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "env", "reg", "a"),
		discovery.TestModule("svc", "env", "reg", "b"),
		discovery.TestModule("svc", "env", "reg", "c"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/a": {},
		"svc/env/reg/b": {DependsOn: []string{"svc/env/reg/a"}},
		"svc/env/reg/c": {DependsOn: []string{"svc/env/reg/b"}},
	}
	g := BuildFromDependencies(modules, deps)

	all := g.GetAllDependencies("svc/env/reg/c")
	if len(all) != 2 {
		t.Errorf("GetAllDependencies(c) = %v, want 2 items", all)
	}
}

func TestGetAllDependents(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("svc", "env", "reg", "a"),
		discovery.TestModule("svc", "env", "reg", "b"),
		discovery.TestModule("svc", "env", "reg", "c"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"svc/env/reg/a": {},
		"svc/env/reg/b": {DependsOn: []string{"svc/env/reg/a"}},
		"svc/env/reg/c": {DependsOn: []string{"svc/env/reg/b"}},
	}
	g := BuildFromDependencies(modules, deps)

	all := g.GetAllDependents("svc/env/reg/a")
	if len(all) != 2 {
		t.Errorf("GetAllDependents(a) = %v, want 2 items", all)
	}
}

func TestSubgraph(t *testing.T) {
	g := buildTestGraph()

	sub := g.Subgraph([]string{
		"platform/stage/eu-central-1/vpc",
		"platform/stage/eu-central-1/eks",
		"platform/stage/eu-central-1/app",
	})

	if len(sub.Nodes()) != 3 {
		t.Errorf("expected 3 nodes in subgraph, got %d", len(sub.Nodes()))
	}

	appDeps := sub.GetDependencies("platform/stage/eu-central-1/app")
	if len(appDeps) != 1 || appDeps[0] != "platform/stage/eu-central-1/eks" {
		t.Errorf("app should depend only on eks in subgraph, got %v", appDeps)
	}
}

func TestScopeToModule(t *testing.T) {
	g := buildTestGraph()

	t.Run("dependencies", func(t *testing.T) {
		sub, err := g.ScopeToModule("platform/stage/eu-central-1/app", false)
		if err != nil {
			t.Fatalf("ScopeToModule() error = %v", err)
		}
		nodes := sub.Nodes()
		// app depends on eks, eks depends on vpc → 3 nodes
		if len(nodes) < 2 {
			t.Errorf("expected >= 2 nodes, got %d", len(nodes))
		}
		if sub.GetNode("platform/stage/eu-central-1/app") == nil {
			t.Error("missing app node")
		}
	})

	t.Run("dependents", func(t *testing.T) {
		sub, err := g.ScopeToModule("platform/stage/eu-central-1/vpc", true)
		if err != nil {
			t.Fatalf("ScopeToModule() error = %v", err)
		}
		if sub.GetNode("platform/stage/eu-central-1/vpc") == nil {
			t.Error("missing vpc node")
		}
		// vpc has dependents (eks, rds, etc.)
		if len(sub.Nodes()) < 2 {
			t.Errorf("expected >= 2 nodes for dependents, got %d", len(sub.Nodes()))
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := g.ScopeToModule("nonexistent/module", false)
		if err == nil {
			t.Error("expected error for nonexistent module")
		}
	})
}

// --- Library usage ---

func TestLibraryUsage(t *testing.T) {
	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-north-1", "vpc"),
		discovery.TestModule("platform", "stage", "eu-north-1", "msk"),
	}
	deps := map[string]*parser.ModuleDependencies{
		"platform/stage/eu-north-1/vpc": {},
		"platform/stage/eu-north-1/msk": {
			DependsOn:           []string{"platform/stage/eu-north-1/vpc"},
			LibraryDependencies: []*parser.LibraryDependency{{LibraryPath: "/project/_modules/kafka"}, {LibraryPath: "/project/_modules/kafka_acl"}},
		},
	}
	g := BuildFromDependencies(modules, deps)

	users := g.GetModulesUsingLibrary("/project/_modules/kafka")
	if len(users) != 1 || users[0] != "platform/stage/eu-north-1/msk" {
		t.Errorf("kafka users = %v, want [msk]", users)
	}
	if paths := g.GetAllLibraryPaths(); len(paths) != 2 {
		t.Errorf("expected 2 library paths, got %d", len(paths))
	}
}

func TestAddLibraryUsage_Dedup(t *testing.T) {
	g := NewDependencyGraph()
	g.AddLibraryUsage("/lib/kafka", "module-a")
	g.AddLibraryUsage("/lib/kafka", "module-a")

	if users := g.GetModulesUsingLibrary("/lib/kafka"); len(users) != 1 {
		t.Errorf("expected 1 user after dedup, got %d", len(users))
	}
}

func TestNormalizeLibraryPath(t *testing.T) {
	tests := []struct{ input, want string }{
		{"/project/_modules/kafka/", "/project/_modules/kafka"},
		{"/project/_modules/kafka", "/project/_modules/kafka"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := normalizeLibraryPath(tt.input); got != tt.want {
			t.Errorf("normalizeLibraryPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
