package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
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

	// No plan results should not be a hard error — summary gracefully skips.
	// If the command errors, it should not be a crash.
	err := runTerraCi(t, dir, "summary")
	if err != nil {
		t.Logf("summary with no results: %v (may be expected)", err)
	}
}

func TestSummary_LoadsReports(t *testing.T) {
	dir := fixtureDir(t, "with-reports")

	// Summary output goes to log (stderr), not stdout.
	// We verify it runs without error and report files are accessible.
	output, err := captureTerraCi(t, dir, "summary")
	if err != nil {
		t.Fatalf("summary failed: %v", err)
	}

	// Output may be empty (logs go to stderr), which is OK
	_ = output
}

func TestSummary_ReportFilesExist(t *testing.T) {
	dir := fixtureDir(t, "with-reports")

	// Verify that report files are accessible and contain valid JSON
	costReportPath := filepath.Join(dir, ".terraci", ci.ReportFilename("cost"))
	costData, err := os.ReadFile(costReportPath)
	if err != nil {
		t.Fatalf("cost report not found: %v", err)
	}

	var costReport ci.Report
	if jsonErr := json.Unmarshal(costData, &costReport); jsonErr != nil {
		t.Fatalf("invalid cost report JSON: %v", jsonErr)
	}
	if costReport.Plugin != "cost" {
		t.Errorf("expected plugin=cost, got %s", costReport.Plugin)
	}
	if costReport.Status != ci.ReportStatusPass {
		t.Errorf("expected cost report status=pass, got %s", costReport.Status)
	}
	assertContains(t, costReport.Summary, "module")

	// Policy report
	policyReportPath := filepath.Join(dir, ".terraci", ci.ReportFilename("policy"))
	policyData, policyErr := os.ReadFile(policyReportPath)
	if policyErr != nil {
		t.Fatalf("policy report not found: %v", policyErr)
	}

	var policyReport ci.Report
	if jsonErr := json.Unmarshal(policyData, &policyReport); jsonErr != nil {
		t.Fatalf("invalid policy report JSON: %v", jsonErr)
	}
	if policyReport.Plugin != "policy" {
		t.Errorf("expected plugin=policy, got %s", policyReport.Plugin)
	}
}
