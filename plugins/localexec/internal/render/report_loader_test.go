package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/planresults"
)

func TestSummaryReportLoader_LoadMissingReportReturnsNil(t *testing.T) {
	t.Parallel()

	report, err := NewSummaryReportLoader(t.TempDir(), "", nil).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if report != nil {
		t.Fatalf("Load() report = %#v, want nil", report)
	}
}

func TestSummaryReportLoader_LoadSummaryReport(t *testing.T) {
	t.Parallel()

	serviceDir := t.TempDir()
	summary := &ci.Report{
		Producer: "summary",
		Title:    "Terraform Plan Summary",
		Status:   ci.ReportStatusWarn,
		Summary:  "1 module: 1 with changes",
	}
	if err := ci.SaveReport(serviceDir, summary); err != nil {
		t.Fatalf("SaveReport() error = %v", err)
	}

	report, err := NewSummaryReportLoader(serviceDir, "", nil).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if report == nil {
		t.Fatal("Load() report = nil, want summary report")
	}
	if report.Producer != "summary" {
		t.Fatalf("Load() plugin = %q, want summary", report.Producer)
	}
	if report.Title != summary.Title {
		t.Fatalf("Load() title = %q, want %q", report.Title, summary.Title)
	}
}

func TestSummaryReportLoader_ResetRemovesStaleSummaryReport(t *testing.T) {
	t.Parallel()

	serviceDir := t.TempDir()
	summary := &ci.Report{
		Producer: "summary",
		Title:    "Terraform Plan Summary",
		Status:   ci.ReportStatusWarn,
		Summary:  "old",
	}
	if err := ci.SaveReport(serviceDir, summary); err != nil {
		t.Fatalf("SaveReport() error = %v", err)
	}

	loader := NewSummaryReportLoader(serviceDir, "", nil)
	if err := loader.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	report, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if report != nil {
		t.Fatalf("Load() report = %#v, want nil", report)
	}
}

func TestSummaryReportLoader_ResetMissingReportIsNoop(t *testing.T) {
	t.Parallel()

	if err := NewSummaryReportLoader(t.TempDir(), "", nil).Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
}

func TestSummaryReportLoader_EmptyServiceDirReturnsNil(t *testing.T) {
	t.Parallel()

	report, err := NewSummaryReportLoader("", "", nil).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if report != nil {
		t.Fatalf("Load() report = %#v, want nil", report)
	}
}

func TestSummaryReportLoader_LoadInvalidReportReturnsError(t *testing.T) {
	t.Parallel()

	serviceDir := t.TempDir()
	reportPath := filepath.Join(serviceDir, ci.ReportFilename("summary"))
	if err := os.WriteFile(reportPath, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	report, err := NewSummaryReportLoader(serviceDir, "", nil).Load()
	if err == nil {
		t.Fatal("Load() error = nil, want decode error")
	}
	if report != nil {
		t.Fatalf("Load() report = %#v, want nil", report)
	}
	if !strings.Contains(err.Error(), "load summary report") {
		t.Fatalf("Load() error = %v, want wrapped load summary report context", err)
	}
}

func TestSummaryReportLoader_LoadSummaryReportWithMatchingProvenance(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	serviceDir := t.TempDir()
	moduleDir := filepath.Join(workDir, "platform", "stage", "eu-central-1", "vpc")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "plan.json"), []byte(`{"format_version":"1.2","terraform_version":"1.6.0","resource_changes":[]}`), 0o644); err != nil {
		t.Fatalf("WriteFile(plan.json) error = %v", err)
	}

	collection, err := planresults.Scan(workDir, []string{"service", "environment", "region", "module"})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	summary := &ci.Report{
		Producer: "summary",
		Title:    "Terraform Plan Summary",
		Status:   ci.ReportStatusPass,
		Summary:  "1 module",
		Provenance: &ci.ReportProvenance{
			PlanResultsFingerprint: collection.Fingerprint(),
		},
	}
	if err = ci.SaveReport(serviceDir, summary); err != nil {
		t.Fatalf("SaveReport() error = %v", err)
	}

	report, err := NewSummaryReportLoader(serviceDir, workDir, []string{"service", "environment", "region", "module"}).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if report == nil {
		t.Fatal("Load() report = nil, want summary report")
	}
}

func TestSummaryReportLoader_LoadSummaryReportWithMismatchedProvenanceReturnsError(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	serviceDir := t.TempDir()
	moduleDir := filepath.Join(workDir, "platform", "stage", "eu-central-1", "vpc")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "plan.json"), []byte(`{"format_version":"1.2","terraform_version":"1.6.0","resource_changes":[]}`), 0o644); err != nil {
		t.Fatalf("WriteFile(plan.json) error = %v", err)
	}

	summary := &ci.Report{
		Producer: "summary",
		Title:    "Terraform Plan Summary",
		Status:   ci.ReportStatusPass,
		Summary:  "1 module",
		Provenance: &ci.ReportProvenance{
			PlanResultsFingerprint: "stale",
		},
	}
	if err := ci.SaveReport(serviceDir, summary); err != nil {
		t.Fatalf("SaveReport() error = %v", err)
	}

	report, err := NewSummaryReportLoader(serviceDir, workDir, []string{"service", "environment", "region", "module"}).Load()
	if err == nil {
		t.Fatal("Load() error = nil, want provenance mismatch")
	}
	if report != nil {
		t.Fatalf("Load() report = %#v, want nil", report)
	}
	if !strings.Contains(err.Error(), "provenance mismatch") {
		t.Fatalf("Load() error = %v, want provenance mismatch", err)
	}
}
