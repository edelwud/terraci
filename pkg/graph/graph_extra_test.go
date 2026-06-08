package graph

import (
	"fmt"
	"testing"

	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/discovery"
)

func TestAddEdgeProducesDiagnosticWhenNodesMissing(t *testing.T) {
	g := NewDependencyGraph()
	g.AddEdge("from", "to")
	d := g.Diagnostics()
	if d.Empty() {
		t.Fatal("expected diagnostics, got none")
	}
	warnings := d.Filter(diagnostic.SeverityWarning)
	if len(warnings) == 0 {
		t.Fatalf("expected at least one warning diagnostic, got %d", len(warnings))
	}
	// Ensure diagnostic references the origin module
	found := false
	for _, w := range warnings {
		if w.Module() == "from" || w.Message() == "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected a diagnostic mentioning module 'from', got: %+v", warnings)
	}
}

func TestTopologicalSortDeterministicAndCorrect(t *testing.T) {
	// Build simple graph: a <- b, a <- c (b and c depend on a)
	g := NewDependencyGraph()
	a := discovery.TestModule("svc", "env", "reg", "a")
	b := discovery.TestModule("svc", "env", "reg", "b")
	c := discovery.TestModule("svc", "env", "reg", "c")
	g.AddNode(a)
	g.AddNode(b)
	g.AddNode(c)
	g.AddEdge(b.ID(), a.ID())
	g.AddEdge(c.ID(), a.ID())

	first, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort error: %v", err)
	}
	second, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort error on second run: %v", err)
	}
	// Deterministic: two runs produce identical order
	if len(first) != len(second) {
		t.Fatalf("topo lengths differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("topo ordering not deterministic: %v vs %v", first, second)
		}
	}
	// Check that 'a' comes before both 'b' and 'c'
	pos := func(id string) int {
		for i, v := range first {
			if v == id {
				return i
			}
		}
		return -1
	}
	pa := pos(a.ID())
	pb := pos(b.ID())
	pc := pos(c.ID())
	if pa == -1 || pb == -1 || pc == -1 {
		t.Fatalf("missing nodes in topo result: %v", first)
	}
	if pa > pb || pa > pc {
		t.Fatalf("dependency ordering violated: a should appear before b and c; got order %v", first)
	}
}

func TestCollectTransitiveIterativeDeepChain(t *testing.T) {
	const n = 1000
	g := NewDependencyGraph()
	// create nodes n0..n(n-1)
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("m%d", i)
		m := discovery.TestModule("svc", "env", "reg", name)
		g.AddNode(m)
		ids[i] = m.ID()
	}
	// add chain edges: m0 -> m1 -> m2 -> ...
	for i := 0; i < n-1; i++ {
		g.AddEdge(ids[i], ids[i+1])
	}

	all := g.GetAllDependencies(ids[0])
	if len(all) != n-1 {
		t.Fatalf("expected %d transitive deps, got %d", n-1, len(all))
	}
	// Ensure last element is m(n-1)
	if all[len(all)-1] != ids[n-1] {
		t.Fatalf("expected last transitive dependency to be %s, got %s", ids[n-1], all[len(all)-1])
	}
}
