package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestPolicy_NotEnabled(t *testing.T) {
	err := runTerraCi(t, fixtureDir(t, "basic"), "policy", "check")
	if err == nil {
		t.Fatal("expected error when policy not enabled")
	}
}

func TestPolicy_PullNotEnabled(t *testing.T) {
	err := runTerraCi(t, fixtureDir(t, "basic"), "policy", "pull")
	if err == nil {
		t.Fatal("expected error when policy not enabled")
	}
}

func TestPolicy_Pull(t *testing.T) {
	dir := copyFixtureToTemp(t, "with-policy")

	// Pull should work (path source copies policies to cache)
	err := runTerraCi(t, dir, "policy", "pull")
	if err != nil {
		t.Fatalf("policy pull failed: %v", err)
	}
}

func TestPolicy_Check(t *testing.T) {
	dir := copyFixtureToTemp(t, "with-policy")

	err := runTerraCi(t, dir, "policy", "check")
	if err != nil {
		t.Fatalf("policy check failed: %v", err)
	}

	// Should have created policy-results.json in service dir
	resultsPath := filepath.Join(dir, ".terraci", "policy-results.json")
	if _, statErr := os.Stat(resultsPath); os.IsNotExist(statErr) {
		t.Fatal("policy-results.json not created")
	}

	// Should have created policy-report.json
	reportPath := filepath.Join(dir, ".terraci", "policy-report.json")
	data, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		t.Fatalf("failed to read policy report: %v", readErr)
	}

	var report ci.Report
	if jsonErr := json.Unmarshal(data, &report); jsonErr != nil {
		t.Fatalf("invalid report JSON: %v", jsonErr)
	}

	if report.Plugin != "policy" {
		t.Errorf("expected plugin=policy, got %s", report.Plugin)
	}

	if report.Status != ci.ReportStatusWarn {
		t.Errorf("expected status=warn (rego has warn rule), got %s", report.Status)
	}
}

func TestPolicy_CheckModule(t *testing.T) {
	dir := copyFixtureToTemp(t, "with-policy")

	err := runTerraCi(t, dir, "policy", "check", "--module", "platform/prod/eu-central-1/vpc")
	if err != nil {
		t.Fatalf("policy check --module failed: %v", err)
	}
}
