package ci

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryReportStore_PublishAndLoadReports(t *testing.T) {
	store := NewMemoryReportStore()
	report := testStoreReport("cost", "Cost", ReportStatusWarn)

	publishStoreReport(t, store, report)
	collection, err := store.LoadReports(context.Background())
	if err != nil {
		t.Fatalf("LoadReports() error = %v", err)
	}
	got, ok := collection.Find("cost")
	if !ok {
		t.Fatal("Find(cost) ok = false, want true")
	}
	if got.Producer() != "cost" || got.Title() != "Cost" || got.Status() != ReportStatusWarn {
		t.Fatalf("Find(cost) = %#v, want original report fields", got)
	}
}

func TestMemoryReportStore_LoadReportsMissing(t *testing.T) {
	store := NewMemoryReportStore()
	collection, err := store.LoadReports(context.Background())
	if err != nil {
		t.Fatalf("LoadReports() error = %v", err)
	}
	if got, ok := collection.Find("missing"); ok || got != nil {
		t.Fatalf("Find(missing) = (%#v, %v), want (nil, false)", got, ok)
	}
}

func TestMemoryReportStore_PublishOverwrite(t *testing.T) {
	store := NewMemoryReportStore()
	publishStoreReport(t, store, testStoreReport("policy", "Old", ReportStatusPass))
	publishStoreReport(t, store, testStoreReport("policy", "New", ReportStatusFail))

	collection, err := store.LoadReports(context.Background())
	if err != nil {
		t.Fatalf("LoadReports() error = %v", err)
	}
	got, ok := collection.Find("policy")
	if !ok {
		t.Fatal("Find(policy) ok = false, want true")
	}
	if got.Title() != "New" || got.Status() != ReportStatusFail {
		t.Fatalf("Find(policy) = %#v, want overwritten report", got)
	}
}

func TestMemoryReportStore_LoadReportsSorted(t *testing.T) {
	store := NewMemoryReportStore()
	publishStoreReport(t, store, testStoreReport("tfupdate", "TF Update", ReportStatusPass))
	publishStoreReport(t, store, testStoreReport("cost", "Cost", ReportStatusWarn))

	collection, err := store.LoadReports(context.Background())
	if err != nil {
		t.Fatalf("LoadReports() error = %v", err)
	}
	reports := collection.Reports()
	if len(reports) != 2 {
		t.Fatalf("Reports() len = %d, want 2", len(reports))
	}
	if reports[0].Producer() != "cost" || reports[1].Producer() != "tfupdate" {
		t.Fatalf("Reports() order = [%s %s], want [cost tfupdate]", reports[0].Producer(), reports[1].Producer())
	}
}

func TestMemoryReportStore_DefensiveCopies(t *testing.T) {
	store := NewMemoryReportStore()
	original := testStoreReport("cost", "Cost", ReportStatusPass)
	publishStoreReport(t, store, original)

	original.title = "mutated original"
	first, err := store.LoadReports(context.Background())
	if err != nil {
		t.Fatalf("LoadReports() error = %v", err)
	}
	got, ok := first.Find("cost")
	if !ok {
		t.Fatal("Find(cost) ok = false, want true")
	}
	got.title = "mutated returned"

	second, err := store.LoadReports(context.Background())
	if err != nil {
		t.Fatalf("LoadReports(second) error = %v", err)
	}
	again, ok := second.Find("cost")
	if !ok {
		t.Fatal("Find(cost) second ok = false, want true")
	}
	if again.Title() != "Cost" {
		t.Fatalf("stored report title = %q, want Cost", again.Title())
	}
}

func TestFileReportStore_LoadReportsMergesMemoryOverlay(t *testing.T) {
	ctx := context.Background()
	store := NewFileReportStore(t.TempDir())
	publishStoreReport(t, store, testStoreReport("cost", "Cost", ReportStatusPass))
	publishStoreReport(t, store, testStoreReport("policy", "Policy", ReportStatusWarn))

	collection, err := store.LoadReports(ctx)
	if err != nil {
		t.Fatalf("LoadReports: %v", err)
	}
	reports := collection.Reports()
	if len(reports) != 2 {
		t.Fatalf("LoadReports() len = %d, want 2", len(reports))
	}
	if reports[0].Producer() != "cost" || reports[1].Producer() != "policy" {
		t.Fatalf("LoadReports() order = [%s %s], want [cost policy]", reports[0].Producer(), reports[1].Producer())
	}
}

func TestFileReportStore_PublishArtifactsAttemptsBothWrites(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := NewFileReportStore(dir)
	report := &Report{producer: "cost", status: ReportStatusPass}

	publication, err := NewArtifactPublication(ArtifactPublicationOptions{
		Producer: "cost",
		Results:  RawResults(map[string]string{"ok": "true"}),
		BuildReport: func() (*Report, error) {
			return report, nil
		},
	})
	if err != nil {
		t.Fatalf("NewArtifactPublication() error = %v", err)
	}
	err = store.PublishArtifacts(ctx, publication)
	if err == nil {
		t.Fatal("PublishArtifacts() error = nil, want invalid report error")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("PublishArtifacts() error = %q, want report validation error", err.Error())
	}
	if _, statErr := os.Stat(filepath.Join(dir, ResultFilename("cost"))); statErr != nil {
		t.Fatalf("results file was not written: %v", statErr)
	}
}

func TestFileReportStore_PublishArtifactsOverwritesReport(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := NewFileReportStore(dir)

	publishStoreArtifacts(t, store, "cost", map[string]string{"version": "old"}, testStoreReport("cost", "Old", ReportStatusPass))
	publishStoreArtifacts(t, store, "cost", map[string]string{"version": "new"}, testStoreReport("cost", "New", ReportStatusWarn))

	collection, err := store.LoadReports(ctx)
	if err != nil {
		t.Fatalf("LoadReports() error = %v", err)
	}
	report, ok := collection.Find("cost")
	if !ok || report.Title() != "New" || report.Status() != ReportStatusWarn {
		t.Fatalf("report = %#v, want overwritten report", report)
	}
	data, err := os.ReadFile(filepath.Join(dir, ResultFilename("cost")))
	if err != nil {
		t.Fatalf("ReadFile(results) error = %v", err)
	}
	if !strings.Contains(string(data), "new") {
		t.Fatalf("results = %s, want overwritten raw results", data)
	}
}

func TestFileReportStore_PublishArtifactsDeletesStaleReport(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store := NewFileReportStore(dir)

	publishStoreReport(t, store, testStoreReport("cost", "Cost", ReportStatusPass))
	publishStoreArtifacts(t, store, "cost", map[string]string{"ok": "true"}, nil)
	if _, err := os.Stat(filepath.Join(dir, ReportFilename("cost"))); !os.IsNotExist(err) {
		t.Fatalf("report file exists after nil replacement: %v", err)
	}
	collection, err := store.LoadReports(ctx)
	if err != nil {
		t.Fatalf("LoadReports() error = %v", err)
	}
	if _, ok := collection.Find("cost"); ok {
		t.Fatal("Find(cost) ok = true after nil replacement, want false")
	}
}

func TestFileReportStore_PublishArtifactsMissingReportDeleteIsNoop(t *testing.T) {
	store := NewFileReportStore(t.TempDir())
	publishStoreArtifacts(t, store, "cost", map[string]string{"ok": "true"}, nil)
}

func testStoreReport(producer, title string, status ReportStatus) *Report {
	return &Report{producer: producer, title: title, status: status}
}

func publishStoreReport(tb testing.TB, store ArtifactPublisher, report *Report) {
	tb.Helper()
	publishStoreArtifacts(tb, store, report.Producer(), NoResults(), report)
}

func publishStoreArtifacts(tb testing.TB, store ArtifactPublisher, producer string, results any, report *Report) {
	tb.Helper()
	artifactResults, ok := results.(ArtifactResults)
	if !ok {
		artifactResults = RawResults(results)
	}
	publication, err := NewArtifactPublication(ArtifactPublicationOptions{
		Producer: producer,
		Results:  artifactResults,
		BuildReport: func() (*Report, error) {
			return report, nil
		},
	})
	if err != nil {
		tb.Fatalf("NewArtifactPublication() error = %v", err)
	}
	if err := store.PublishArtifacts(context.Background(), publication); err != nil {
		tb.Fatalf("PublishArtifacts(%s) error = %v", producer, err)
	}
}
