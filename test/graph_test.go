package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraph_DOTFormat(t *testing.T) {
	dir := fixtureDir(t, "basic")
	outFile := filepath.Join(t.TempDir(), "graph.dot")

	err := runTerraCi(t, dir, "graph", "--format", "dot", "-o", outFile)
	if err != nil {
		t.Fatalf("graph --format dot failed: %v", err)
	}

	data, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatalf("failed to read output: %v", readErr)
	}
	output := string(data)

	assertContains(t, output, "digraph")

	// Should reference module paths (4 modules: vpc x2, eks, app)
	assertContains(t, output, "vpc")
	assertContains(t, output, "eks")
	assertContains(t, output, "app")

	// Should contain dependency edges
	assertContains(t, output, "->")
}

func TestGraph_DOTToStdout(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "dot")
	if err != nil {
		t.Fatalf("graph failed: %v", err)
	}

	assertContains(t, output, "digraph")
	assertContains(t, output, "vpc")
	assertContains(t, output, "eks")
	assertContains(t, output, "app")
	assertContains(t, output, "->")
}

func TestGraph_LevelsFormat(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "levels")
	if err != nil {
		t.Fatalf("graph --format levels failed: %v", err)
	}

	assertContains(t, output, "Level 0")

	// Basic fixture has dependencies, so there should be multiple levels
	assertContains(t, output, "Level 1")
}

func TestGraph_ListFormat(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "list")
	if err != nil {
		t.Fatalf("graph --format list failed: %v", err)
	}

	if output == "" {
		t.Error("expected non-empty list output")
	}

	// List format uses grouped output: [service/env] prefix with module paths
	assertContains(t, output, "platform/prod")
	assertContains(t, output, "platform/stage")
	assertContains(t, output, "vpc")
	assertContains(t, output, "eks")

	// Should show dependency arrows for modules with deps
	assertContains(t, output, "→")
}

func TestGraph_Stats(t *testing.T) {
	dir := fixtureDir(t, "basic")

	// --stats outputs via log (stderr), not stdout, so we cannot capture the content.
	// The key assertion is that the command succeeds, which means the graph was built
	// and stats were computed without errors.
	err := runTerraCi(t, dir, "graph", "--stats")
	if err != nil {
		t.Fatalf("graph --stats failed: %v", err)
	}
}

func TestGraph_InvalidDir(t *testing.T) {
	dir := t.TempDir()
	err := runTerraCi(t, dir, "graph", "--format", "dot")
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no modules found") {
		t.Errorf("expected 'no modules found' error, got: %v", err)
	}
}

func TestGraph_PlantUMLFormat(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "plantuml")
	if err != nil {
		t.Fatalf("graph --format plantuml failed: %v", err)
	}

	assertContains(t, output, "@startuml")
	assertContains(t, output, "@enduml")
	assertContains(t, output, "-->")
	assertContains(t, output, "vpc")
	assertContains(t, output, "eks")
}

func TestGraph_ModuleScope(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "list", "--module", "platform/prod/eu-central-1/vpc")
	if err != nil {
		t.Fatalf("graph --module failed: %v", err)
	}

	assertContains(t, output, "vpc")
}

func TestGraph_ModuleDependents(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "list", "--module", "platform/prod/eu-central-1/vpc", "--dependents")
	if err != nil {
		t.Fatalf("graph --module --dependents failed: %v", err)
	}

	// vpc and its dependent eks should appear
	assertContains(t, output, "vpc")
	assertContains(t, output, "eks")
}

func TestGraph_ExcludeFilter(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "levels", "--exclude", "**/eks")
	if err != nil {
		t.Fatalf("graph --exclude failed: %v", err)
	}

	assertContains(t, output, "vpc")
	assertNotContains(t, output, "eks")
}

func TestGraph_IncludeFilter(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "levels", "--include", "**/vpc")
	if err != nil {
		t.Fatalf("graph --include failed: %v", err)
	}

	assertContains(t, output, "vpc")
	assertNotContains(t, output, "eks")
	assertNotContains(t, output, "app")
}

func TestGraph_InvalidFormat(t *testing.T) {
	dir := fixtureDir(t, "basic")

	err := runTerraCi(t, dir, "graph", "--format", "csv")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestGraph_DOTEdgeCorrectness(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "dot")
	if err != nil {
		t.Fatalf("graph --format dot failed: %v", err)
	}

	// eks depends on vpc, so there should be an edge eks -> vpc
	assertContains(t, output, "digraph")

	// Check that the DOT output contains an edge representing eks -> vpc dependency
	hasEksVpcEdge := false
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "eks") && strings.Contains(trimmed, "->") && strings.Contains(trimmed, "vpc") {
			hasEksVpcEdge = true
			break
		}
	}
	if !hasEksVpcEdge {
		t.Error("DOT output should contain an edge from eks to vpc")
	}
}
