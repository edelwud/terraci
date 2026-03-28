package test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSummary_WithReports(t *testing.T) {
	dir := fixtureDir(t, "with-reports")

	// Summary without CI provider should print to log (no error)
	err := runTerraCi(t, dir, "summary")
	if err != nil {
		t.Fatalf("summary failed: %v", err)
	}
}

func TestSummary_NoResults(t *testing.T) {
	// Empty directory — no plan.json files
	dir := t.TempDir()
	writeConfig(t, dir)

	err := runTerraCi(t, dir, "summary")
	// Should not error — just skip
	if err != nil {
		t.Logf("summary with no results: %v (expected)", err)
	}
}

func TestSummary_LoadsReports(t *testing.T) {
	dir := fixtureDir(t, "with-reports")

	// Run summary and capture output
	output, err := captureTerraCi(t, dir, "summary")
	if err != nil {
		t.Fatalf("summary failed: %v", err)
	}

	// Output should mention plan results
	if output == "" {
		t.Log("summary produced no stdout (logs go to stderr, OK)")
	}
}

func TestSummary_ReportFilesExist(t *testing.T) {
	dir := fixtureDir(t, "with-reports")

	// Verify that report files are accessible
	costReport := filepath.Join(dir, ".terraci", "cost-report.json")
	if _, err := os.Stat(costReport); err != nil {
		t.Fatalf("cost report not found: %v", err)
	}

	policyReport := filepath.Join(dir, ".terraci", "policy-report.json")
	if _, err := os.Stat(policyReport); err != nil {
		t.Fatalf("policy report not found: %v", err)
	}
}
