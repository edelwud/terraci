package ci

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveReport(t *testing.T) {
	dir := t.TempDir()
	report := &Report{
		Plugin:  "test",
		Title:   "Test Report",
		Status:  ReportStatusPass,
		Summary: "all good",
		Sections: []ReportSection{{
			Kind:           ReportSectionKindOverview,
			Title:          "Summary",
			Status:         ReportStatusPass,
			SectionSummary: "all good",
			Overview: &OverviewSection{
				PlanStats: SummaryPlanStats{Total: 1, NoChanges: 1, Success: 1},
			},
		}},
	}

	if err := SaveReport(dir, report); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	path := filepath.Join(dir, ReportFilename("test"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var loaded Report
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.Plugin != "test" {
		t.Errorf("plugin = %q, want test", loaded.Plugin)
	}
	if loaded.Status != ReportStatusPass {
		t.Errorf("status = %q, want pass", loaded.Status)
	}
}

func TestSaveJSON(t *testing.T) {
	dir := t.TempDir()
	data := map[string]string{"key": "value"}

	if err := SaveJSON(dir, "test.json", data); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "test.json"))
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

func TestSaveJSON_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	data := map[string]string{"a": "b"}

	if err := SaveJSON(dir, "test.json", data); err != nil {
		t.Fatalf("SaveJSON with nested dir: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "test.json"))
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
	report := &Report{
		Plugin: "cost",
		Title:  "Cost Report",
		Status: ReportStatusWarn,
		Sections: []ReportSection{{
			Kind:           ReportSectionKindCostChanges,
			Title:          "Cost Estimation",
			Status:         ReportStatusWarn,
			SectionSummary: "1 module",
			CostChanges: &CostChangesSection{
				Totals: CostTotals{After: 15, Diff: 5},
				Rows: []CostChangeRow{
					{ModulePath: "svc/prod/eu/vpc", Before: 10.0, After: 15.0, Diff: 5.0, HasCost: true},
				},
			},
		}},
	}

	if err := SaveReport(dir, report); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ReportFilename("cost")))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var loaded Report
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(loaded.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(loaded.Sections))
	}
	if loaded.Sections[0].CostChanges == nil {
		t.Fatal("expected cost section payload")
	}
	if loaded.Sections[0].CostChanges.Rows[0].Diff != 5.0 {
		t.Errorf("cost diff = %f, want 5.0", loaded.Sections[0].CostChanges.Rows[0].Diff)
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
		{name: "missing plugin", report: &Report{Title: "Missing Plugin", Status: ReportStatusPass}, wantErr: "plugin is required"},
		{name: "unsafe plugin name", report: &Report{Plugin: "../cost", Title: "Cost", Status: ReportStatusPass}, wantErr: "not a safe artifact name"},
		{name: "missing title", report: &Report{Plugin: "cost", Status: ReportStatusPass}, wantErr: "title is required"},
		{name: "invalid status", report: &Report{Plugin: "cost", Title: "Cost", Status: "unknown"}, wantErr: `status "unknown" is invalid`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := SaveReport(t.TempDir(), tt.report)
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
	if got := ReportFilename("cost"); got != "cost-report.json" {
		t.Fatalf("ReportFilename(cost) = %q, want cost-report.json", got)
	}
}

func TestLoadReport(t *testing.T) {
	dir := t.TempDir()
	report := &Report{
		Plugin:  "policy",
		Title:   "Policy Check",
		Status:  ReportStatusWarn,
		Summary: "warned",
	}

	if err := SaveReport(dir, report); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	loaded, err := LoadReport(filepath.Join(dir, ReportFilename("policy")))
	if err != nil {
		t.Fatalf("LoadReport: %v", err)
	}

	if loaded.Plugin != "policy" {
		t.Fatalf("plugin = %q, want policy", loaded.Plugin)
	}
	if loaded.Status != ReportStatusWarn {
		t.Fatalf("status = %q, want warn", loaded.Status)
	}
}

func TestLoadReports(t *testing.T) {
	dir := t.TempDir()
	reports := []*Report{
		{Plugin: "update", Title: "Update", Status: ReportStatusPass},
		{Plugin: "cost", Title: "Cost", Status: ReportStatusWarn},
	}

	for _, report := range reports {
		if err := SaveReport(dir, report); err != nil {
			t.Fatalf("SaveReport(%s): %v", report.Plugin, err)
		}
	}

	loaded, err := LoadReports(dir)
	if err != nil {
		t.Fatalf("LoadReports: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("loaded report count = %d, want 2", len(loaded))
	}
	if loaded[0].Plugin != "cost" || loaded[1].Plugin != "update" {
		t.Fatalf("loaded report order = [%s %s], want [cost update]", loaded[0].Plugin, loaded[1].Plugin)
	}
}

func TestLoadReport_UnknownSectionKindFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ReportFilename("broken"))
	content := `{
  "plugin": "broken",
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
	}
}

func TestLoadReport_InvalidRootFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ReportFilename("broken"))
	content := `{
  "plugin": "broken",
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
	report := &Report{
		Plugin:  "summary",
		Title:   "Terraform Plan Summary",
		Status:  ReportStatusWarn,
		Summary: "summary",
		Provenance: &ReportProvenance{
			Producer:               "summary",
			GeneratedAt:            time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC),
			CommitSHA:              "abcdef1234567890",
			PipelineID:             "123",
			PlanResultsFingerprint: "fingerprint",
		},
	}

	if err := SaveReport(dir, report); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	loaded, err := LoadReport(filepath.Join(dir, ReportFilename("summary")))
	if err != nil {
		t.Fatalf("LoadReport: %v", err)
	}
	if loaded.Provenance == nil {
		t.Fatal("Provenance = nil, want value")
	}
	if loaded.Provenance.Producer != "summary" {
		t.Fatalf("Producer = %q, want summary", loaded.Provenance.Producer)
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
