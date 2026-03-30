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
	assertContains(t, err.Error(), "not enabled")
}

func TestPolicy_PullNotEnabled(t *testing.T) {
	err := runTerraCi(t, fixtureDir(t, "basic"), "policy", "pull")
	if err == nil {
		t.Fatal("expected error when policy not enabled")
	}
	assertContains(t, err.Error(), "not enabled")
}

func TestPolicy_Pull(t *testing.T) {
	dir := copyFixtureToTemp(t, "with-policy")

	// Pull should work (path source copies policies to cache)
	err := runTerraCi(t, dir, "policy", "pull")
	if err != nil {
		t.Fatalf("policy pull failed: %v", err)
	}

	// Verify policy cache directory was created.
	// For path sources, policies are referenced in-place (not copied to cache),
	// so the cache dir is created but may be empty. The important thing is
	// that pull succeeded and the cache dir exists.
	cacheDir := filepath.Join(dir, ".terraci", "policies")
	if _, statErr := os.Stat(cacheDir); os.IsNotExist(statErr) {
		t.Error("policy cache directory should be created after pull")
	}

	// Verify the original policy files still exist (path source references them)
	policyFile := filepath.Join(dir, "policies", "terraform.rego")
	if _, statErr := os.Stat(policyFile); os.IsNotExist(statErr) {
		t.Error("policy source file should exist after pull")
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

	// Validate policy-results.json content
	resultsData, readResultsErr := os.ReadFile(resultsPath)
	if readResultsErr != nil {
		t.Fatalf("failed to read policy results: %v", readResultsErr)
	}
	assertContains(t, string(resultsData), "warn")

	// Should have created policy-report.json
	reportPath := filepath.Join(dir, ".terraci", ci.ReportFilename("policy"))
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

	// Our .rego rule warns on create actions — plan has a VPC create
	if report.Status != ci.ReportStatusWarn {
		t.Errorf("expected status=warn (rego has warn rule), got %s", report.Status)
	}

	// Summary should mention warnings
	assertContains(t, report.Summary, "warned")
}

func TestPolicy_CheckModule(t *testing.T) {
	dir := copyFixtureToTemp(t, "with-policy")

	err := runTerraCi(t, dir, "policy", "check", "--module", "platform/prod/eu-central-1/vpc")
	if err != nil {
		t.Fatalf("policy check --module failed: %v", err)
	}

	// Verify results were written for the specific module
	resultsPath := filepath.Join(dir, ".terraci", "policy-results.json")
	data, readErr := os.ReadFile(resultsPath)
	if readErr != nil {
		t.Fatalf("failed to read policy results: %v", readErr)
	}
	assertContains(t, string(data), "vpc")
}
