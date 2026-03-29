package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCost_NotEnabled(t *testing.T) {
	// basic fixture has no cost config — should error
	err := runTerraCi(t, fixtureDir(t, "basic"), "cost")
	if err == nil {
		t.Fatal("expected error when cost not enabled")
	}
	// Verify specific error message, not just any error
	assertContains(t, err.Error(), "not enabled")
}

func TestCost_NoPlanFiles(t *testing.T) {
	// temp dir with cost enabled but no plan.json
	dir := t.TempDir()
	writeCostConfig(t, dir)

	err := runTerraCi(t, dir, "cost")
	if err == nil {
		t.Fatal("expected error when no plan.json files")
	}
	assertContains(t, err.Error(), "no plan.json")
}

func TestCost_WithPlanFiles(t *testing.T) {
	// Copy fixture to temp dir so we don't modify originals
	dir := copyFixtureToTemp(t, "with-cost")

	// Cost estimation will attempt AWS pricing API which is unavailable in test,
	// but the command should at least find plan.json files and proceed past that check.
	err := runTerraCi(t, dir, "cost")
	if err == nil {
		// If it somehow succeeds (e.g., cached pricing), that's fine too
		return
	}

	// Should NOT be "no plan.json files found" — we have one
	if strings.Contains(err.Error(), "no plan.json") {
		t.Fatal("plan.json should have been found in fixture")
	}

	// Any other error is expected (pricing API failure, etc.)
	t.Logf("cost estimation error (expected without AWS): %v", err)
}

func TestCost_ModuleFilter(t *testing.T) {
	dir := copyFixtureToTemp(t, "with-cost")

	// Non-existent module path should result in "no plan.json files found"
	err := runTerraCi(t, dir, "cost", "--module", "nonexistent/path")
	if err == nil {
		t.Fatal("expected error for non-existent module")
	}
	assertContains(t, err.Error(), "no plan.json")
}

func writeCostConfig(t *testing.T, dir string) {
	t.Helper()
	cfg := `structure:
  pattern: "{service}/{environment}/{region}/{module}"
plugins:
  cost:
    providers:
      aws:
        enabled: true
`
	if err := os.WriteFile(filepath.Join(dir, ".terraci.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}
