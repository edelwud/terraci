package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestSummaryReportLoader_LoadMissingReportReturnsNil(t *testing.T) {
	t.Parallel()

	report, err := NewSummaryReportLoader(t.TempDir()).Load()
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
		Plugin:  "summary",
		Title:   "Terraform Plan Summary",
		Summary: "1 module: 1 with changes",
	}
	if err := ci.SaveReport(serviceDir, summary); err != nil {
		t.Fatalf("SaveReport() error = %v", err)
	}

	report, err := NewSummaryReportLoader(serviceDir).Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if report == nil {
		t.Fatal("Load() report = nil, want summary report")
	}
	if report.Plugin != "summary" {
		t.Fatalf("Load() plugin = %q, want summary", report.Plugin)
	}
	if report.Title != summary.Title {
		t.Fatalf("Load() title = %q, want %q", report.Title, summary.Title)
	}
}

func TestSummaryReportLoader_EmptyServiceDirReturnsNil(t *testing.T) {
	t.Parallel()

	report, err := NewSummaryReportLoader("").Load()
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

	report, err := NewSummaryReportLoader(serviceDir).Load()
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
