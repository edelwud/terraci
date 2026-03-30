package ci

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveReport(t *testing.T) {
	dir := t.TempDir()
	report := &Report{
		Plugin:  "test",
		Title:   "Test Report",
		Status:  ReportStatusPass,
		Summary: "all good",
		Body:    "details here",
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

func TestSaveReport_ModulesField(t *testing.T) {
	dir := t.TempDir()
	report := &Report{
		Plugin: "cost",
		Title:  "Cost Report",
		Status: ReportStatusWarn,
		Modules: []ModuleReport{
			{ModulePath: "svc/prod/eu/vpc", CostBefore: 10.0, CostAfter: 15.0, CostDiff: 5.0, HasCost: true},
		},
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

	if len(loaded.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(loaded.Modules))
	}
	if loaded.Modules[0].CostDiff != 5.0 {
		t.Errorf("cost diff = %f, want 5.0", loaded.Modules[0].CostDiff)
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
