package ci

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryReportStore_PublishAndGet(t *testing.T) {
	store := NewMemoryReportStore()
	report := &Report{Producer: "cost", Title: "Cost", Status: ReportStatusWarn}

	store.Publish(report)
	got, ok := store.Get("cost")
	if !ok {
		t.Fatal("Get(cost) ok = false, want true")
	}
	if got.Producer != "cost" || got.Title != "Cost" || got.Status != ReportStatusWarn {
		t.Fatalf("Get(cost) = %#v, want original report fields", got)
	}
}

func TestMemoryReportStore_GetMissing(t *testing.T) {
	store := NewMemoryReportStore()
	if got, ok := store.Get("missing"); ok || got != nil {
		t.Fatalf("Get(missing) = (%#v, %v), want (nil, false)", got, ok)
	}
}

func TestMemoryReportStore_PublishOverwrite(t *testing.T) {
	store := NewMemoryReportStore()
	store.Publish(&Report{Producer: "policy", Title: "Old", Status: ReportStatusPass})
	store.Publish(&Report{Producer: "policy", Title: "New", Status: ReportStatusFail})

	got, ok := store.Get("policy")
	if !ok {
		t.Fatal("Get(policy) ok = false, want true")
	}
	if got.Title != "New" || got.Status != ReportStatusFail {
		t.Fatalf("Get(policy) = %#v, want overwritten report", got)
	}
}

func TestMemoryReportStore_AllSorted(t *testing.T) {
	store := NewMemoryReportStore()
	store.Publish(&Report{Producer: "tfupdate", Title: "TF Update", Status: ReportStatusPass})
	store.Publish(&Report{Producer: "cost", Title: "Cost", Status: ReportStatusWarn})

	reports := store.All()
	if len(reports) != 2 {
		t.Fatalf("All() len = %d, want 2", len(reports))
	}
	if reports[0].Producer != "cost" || reports[1].Producer != "tfupdate" {
		t.Fatalf("All() order = [%s %s], want [cost tfupdate]", reports[0].Producer, reports[1].Producer)
	}
}

func TestMemoryReportStore_DefensiveCopies(t *testing.T) {
	store := NewMemoryReportStore()
	original := &Report{Producer: "cost", Title: "Cost", Status: ReportStatusPass}
	store.Publish(original)

	original.Title = "mutated original"
	got, ok := store.Get("cost")
	if !ok {
		t.Fatal("Get(cost) ok = false, want true")
	}
	got.Title = "mutated returned"

	again, ok := store.Get("cost")
	if !ok {
		t.Fatal("Get(cost) second ok = false, want true")
	}
	if again.Title != "Cost" {
		t.Fatalf("stored report title = %q, want Cost", again.Title)
	}
}

func TestFileReportStore_LoadReportsMergesMemoryOverlay(t *testing.T) {
	ctx := context.Background()
	store := NewFileReportStore(t.TempDir())
	if err := store.SaveReport(ctx, &Report{Producer: "cost", Title: "Cost", Status: ReportStatusPass}); err != nil {
		t.Fatalf("SaveReport(cost): %v", err)
	}
	store.Publish(&Report{Producer: "policy", Title: "Policy", Status: ReportStatusWarn})

	reports, err := store.LoadReports(ctx)
	if err != nil {
		t.Fatalf("LoadReports: %v", err)
	}
	if len(reports) != 2 {
		t.Fatalf("LoadReports() len = %d, want 2", len(reports))
	}
	if reports[0].Producer != "cost" || reports[1].Producer != "policy" {
		t.Fatalf("LoadReports() order = [%s %s], want [cost policy]", reports[0].Producer, reports[1].Producer)
	}
}

func TestFileReportStore_SaveResultsAndReportAttemptsBothWrites(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := NewFileReportStore(dir)
	report := &Report{Producer: "cost", Status: ReportStatusPass}

	err := store.SaveResultsAndReport(ctx, "cost", map[string]string{"ok": "true"}, report)
	if err == nil {
		t.Fatal("SaveResultsAndReport() error = nil, want invalid report error")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("SaveResultsAndReport() error = %q, want report validation error", err.Error())
	}
	if _, statErr := os.Stat(filepath.Join(dir, ResultFilename("cost"))); statErr != nil {
		t.Fatalf("results file was not written: %v", statErr)
	}
}
