package graph

import "testing"

func TestGetStats(t *testing.T) {
	g := buildTestGraph()
	stats := g.GetStats()

	if stats.TotalModules != 4 {
		t.Errorf("TotalModules = %d, want 4", stats.TotalModules)
	}
	if stats.TotalEdges != 4 {
		t.Errorf("TotalEdges = %d, want 4", stats.TotalEdges)
	}
	if stats.RootModules != 1 {
		t.Errorf("RootModules = %d, want 1", stats.RootModules)
	}
	if stats.LeafModules != 1 {
		t.Errorf("LeafModules = %d, want 1", stats.LeafModules)
	}
	if stats.MaxDepth != 2 {
		t.Errorf("MaxDepth = %d, want 2", stats.MaxDepth)
	}
	if stats.HasCycles {
		t.Error("HasCycles should be false")
	}
}

func TestGetStats_Empty(t *testing.T) {
	g := NewDependencyGraph()
	stats := g.GetStats()

	if stats.TotalModules != 0 {
		t.Errorf("TotalModules = %d, want 0", stats.TotalModules)
	}
	if stats.TotalEdges != 0 {
		t.Errorf("TotalEdges = %d, want 0", stats.TotalEdges)
	}
}
