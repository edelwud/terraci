package ci

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileReportStore_SaveReport(t *testing.T) {
	dir := t.TempDir()
	store := NewFileReportStore(dir)
	report, err := NewRenderedReport(RenderedReportOptions{
		Producer: "test",
		Title:    "Test Report",
		Status:   ReportStatusPass,
		Summary:  "all good",
		Sections: []RenderedSectionOptions{{
			Title:   "Summary",
			Summary: "all good",
			Blocks:  []RenderBlock{NewTextBlock(RenderText("1 module analyzed"))},
		}},
	})
	if err != nil {
		t.Fatalf("NewRenderedReport: %v", err)
	}

	if saveErr := store.SaveReport(context.Background(), report); saveErr != nil {
		t.Fatalf("SaveReport: %v", saveErr)
	}

	path := filepath.Join(dir, ReportFilename("test"))
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}

	var loaded Report
	if decodeErr := json.Unmarshal(data, &loaded); decodeErr != nil {
		t.Fatalf("unmarshal: %v", decodeErr)
	}

	if loaded.Producer != "test" {
		t.Errorf("producer = %q, want test", loaded.Producer)
	}
	if loaded.Status != ReportStatusPass {
		t.Errorf("status = %q, want pass", loaded.Status)
	}
}

func TestFileReportStore_SaveResults(t *testing.T) {
	dir := t.TempDir()
	store := NewFileReportStore(dir)
	data := map[string]string{"key": "value"}

	if err := store.SaveResults(context.Background(), "test", data); err != nil {
		t.Fatalf("SaveResults: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ResultFilename("test")))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var loaded map[string]string
	if err := json.Unmarshal(content, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded["key"] != "value" {
		t.Errorf("key = %q, want value", loaded["key"])
	}
}

func TestFileReportStore_SaveResultsCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	store := NewFileReportStore(dir)
	data := map[string]string{"a": "b"}

	if err := store.SaveResults(context.Background(), "test", data); err != nil {
		t.Fatalf("SaveResults with nested dir: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ResultFilename("test")))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var loaded map[string]string
	if err := json.Unmarshal(content, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loaded["a"] != "b" {
		t.Errorf("a = %q, want b", loaded["a"])
	}
}

func TestSaveReport_SectionsField(t *testing.T) {
	dir := t.TempDir()
	store := NewFileReportStore(dir)
	report, err := NewRenderedReport(RenderedReportOptions{
		Producer: "report_a",
		Title:    "Section Report",
		Status:   ReportStatusWarn,
		Sections: []RenderedSectionOptions{{
			Title:   "Sample Section",
			Summary: "1 module",
			Blocks: []RenderBlock{
				NewTableBlock("", []RenderColumn{NewRenderColumn("Module")}, []RenderRow{
					NewRenderRow(RenderModulePath("svc/prod/eu/vpc")),
				}),
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewRenderedReport: %v", err)
	}

	if saveErr := store.SaveReport(context.Background(), report); saveErr != nil {
		t.Fatalf("SaveReport: %v", saveErr)
	}

	data, err := os.ReadFile(filepath.Join(dir, ReportFilename("report_a")))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var loaded Report
	if decodeErr := json.Unmarshal(data, &loaded); decodeErr != nil {
		t.Fatalf("unmarshal: %v", decodeErr)
	}

	if len(loaded.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(loaded.Sections))
	}
	payload, err := DecodeRenderSection(loaded.Sections[0])
	if err != nil {
		t.Fatalf("DecodeRenderSection: %v", err)
	}
	blocks := payload.Blocks()
	rows := blocks[0].Table().Rows()
	if got := rows[0].Cells()[0].Text(); got != "svc/prod/eu/vpc" {
		t.Fatalf("payload module = %q, want svc/prod/eu/vpc", got)
	}
}

func TestSaveReport_RejectsInvalidReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		report  *Report
		wantErr string
	}{
		{name: "nil report", report: nil, wantErr: "ci report is nil"},
		{name: "missing producer", report: &Report{Title: "Missing Producer", Status: ReportStatusPass}, wantErr: "producer is required"},
		{name: "unsafe producer name", report: &Report{Producer: "../report_a", Title: "Cost", Status: ReportStatusPass}, wantErr: "not a safe artifact name"},
		{name: "missing title", report: &Report{Producer: "report_a", Status: ReportStatusPass}, wantErr: "title is required"},
		{name: "invalid status", report: &Report{Producer: "report_a", Title: "Cost", Status: "unknown"}, wantErr: `status "unknown" is invalid`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := NewFileReportStore(t.TempDir())

			err := store.SaveReport(context.Background(), tt.report)
			if err == nil {
				t.Fatal("SaveReport() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("SaveReport() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestReportFilename(t *testing.T) {
	if got := ReportFilename("report_a"); got != "report_a-report.json" {
		t.Fatalf("ReportFilename(report_a) = %q, want report_a-report.json", got)
	}
	if got := ResultFilename("report_a"); got != "report_a-results.json" {
		t.Fatalf("ResultFilename(report_a) = %q, want report_a-results.json", got)
	}
}

func TestLoadReport(t *testing.T) {
	dir := t.TempDir()
	store := NewFileReportStore(dir)
	report := &Report{
		Producer: "report_b",
		Title:    "Report B",
		Status:   ReportStatusWarn,
		Summary:  "warned",
	}

	if err := store.SaveReport(context.Background(), report); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	loaded, err := LoadReport(filepath.Join(dir, ReportFilename("report_b")))
	if err != nil {
		t.Fatalf("LoadReport: %v", err)
	}

	if loaded.Producer != "report_b" {
		t.Fatalf("producer = %q, want report_b", loaded.Producer)
	}
	if loaded.Status != ReportStatusWarn {
		t.Fatalf("status = %q, want warn", loaded.Status)
	}
}

func TestLoadReports(t *testing.T) {
	dir := t.TempDir()
	store := NewFileReportStore(dir)
	reports := []*Report{
		{Producer: "report_c", Title: "Update", Status: ReportStatusPass},
		{Producer: "report_a", Title: "Cost", Status: ReportStatusWarn},
	}

	for _, report := range reports {
		if err := store.SaveReport(context.Background(), report); err != nil {
			t.Fatalf("SaveReport(%s): %v", report.Producer, err)
		}
	}

	loaded, err := store.LoadReports(context.Background())
	if err != nil {
		t.Fatalf("LoadReports: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("loaded report count = %d, want 2", len(loaded))
	}
	if loaded[0].Producer != "report_a" || loaded[1].Producer != "report_c" {
		t.Fatalf("loaded report order = [%s %s], want [report_a report_c]", loaded[0].Producer, loaded[1].Producer)
	}
}

func TestLoadReport_UnknownSectionKindFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ReportFilename("broken"))
	content := `{
  "producer": "broken",
  "title": "Broken",
  "status": "warn",
  "summary": "bad",
  "sections": [
    {
      "kind": "mystery"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := LoadReport(path); err == nil {
		t.Fatal("expected LoadReport to fail for unknown section kind")
	} else if !strings.Contains(err.Error(), "producer reports must use") {
		t.Fatalf("LoadReport() error = %q, want render-ready contract message", err.Error())
	}
}

func TestLoadReport_InvalidRootFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ReportFilename("broken"))
	content := `{
  "producer": "broken",
  "title": "Broken",
  "status": "mystery"
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := LoadReport(path); err == nil {
		t.Fatal("expected LoadReport to fail for invalid root fields")
	}
}

func TestSaveReport_PreservesProvenance(t *testing.T) {
	dir := t.TempDir()
	store := NewFileReportStore(dir)
	report := &Report{
		Producer: "report_c",
		Title:    "Terraform Plan Summary",
		Status:   ReportStatusWarn,
		Summary:  "report_c",
		Provenance: &ReportProvenance{
			GeneratedAt:            time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC),
			CommitSHA:              "abcdef1234567890",
			PipelineID:             "123",
			PlanResultsFingerprint: "fingerprint",
		},
	}

	if err := store.SaveReport(context.Background(), report); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	loaded, err := LoadReport(filepath.Join(dir, ReportFilename("report_c")))
	if err != nil {
		t.Fatalf("LoadReport: %v", err)
	}
	if loaded.Provenance == nil {
		t.Fatal("Provenance = nil, want value")
	}
	if loaded.Producer != "report_c" {
		t.Fatalf("Producer = %q, want report_c", loaded.Producer)
	}
	if loaded.Provenance.PlanResultsFingerprint != "fingerprint" {
		t.Fatalf("PlanResultsFingerprint = %q, want fingerprint", loaded.Provenance.PlanResultsFingerprint)
	}
}

func TestPlanStatusFromPlan(t *testing.T) {
	if PlanStatusFromPlan(false) != PlanStatusNoChanges {
		t.Error("no changes should return PlanStatusNoChanges")
	}
	if PlanStatusFromPlan(true) != PlanStatusChanges {
		t.Error("changes should return PlanStatusChanges")
	}
}

func TestReportStatus_Values(t *testing.T) {
	if ReportStatusPass != "pass" {
		t.Error("pass")
	}
	if ReportStatusWarn != "warn" {
		t.Error("warn")
	}
	if ReportStatusFail != "fail" {
		t.Error("fail")
	}
}
