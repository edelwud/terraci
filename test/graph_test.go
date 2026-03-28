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
