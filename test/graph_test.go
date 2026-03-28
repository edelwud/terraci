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

	if !strings.Contains(output, "digraph") {
		t.Error("expected DOT format with digraph keyword")
	}

	// Should reference module paths
	if !strings.Contains(output, "vpc") {
		t.Error("graph missing vpc module")
	}
}

func TestGraph_DOTToStdout(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "dot")
	if err != nil {
		t.Fatalf("graph failed: %v", err)
	}

	if !strings.Contains(output, "digraph") {
		t.Error("expected DOT format output on stdout")
	}
}

func TestGraph_LevelsFormat(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "levels")
	if err != nil {
		t.Fatalf("graph --format levels failed: %v", err)
	}

	if !strings.Contains(output, "Level 0") {
		t.Error("expected Level 0 in levels output")
	}
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
}

func TestGraph_Stats(t *testing.T) {
	dir := fixtureDir(t, "basic")

	// --stats outputs via log, not stdout, so just check it doesn't error
	err := runTerraCi(t, dir, "graph", "--stats")
	if err != nil {
		t.Fatalf("graph --stats failed: %v", err)
	}
}
