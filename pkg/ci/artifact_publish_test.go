package ci

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishArtifactsWritesResultsAndReport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewFileReportStore(dir)
	artifact := NewArtifactContext(ArtifactContextOptions{})

	err := publishTestArtifacts(store, ArtifactPublicationOptions{
		Producer: "cost",
		Results:  RawResults(map[string]string{"ok": "true"}),
		BuildReport: func() (*Report, error) {
			return testPublishReport(t, "cost", artifact), nil
		},
	})
	if err != nil {
		t.Fatalf("PublishArtifacts() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ResultFilename("cost"))); err != nil {
		t.Fatalf("result file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ReportFilename("cost"))); err != nil {
		t.Fatalf("report file missing: %v", err)
	}
}

func TestPublishArtifactsDeletesStaleReportOnNilReport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewFileReportStore(dir)
	if err := publishTestArtifacts(store, ArtifactPublicationOptions{
		Producer: "cost",
		Results:  NoResults(),
		BuildReport: func() (*Report, error) {
			return testPublishReport(t, "cost", ArtifactContext{}), nil
		},
	}); err != nil {
		t.Fatalf("seed report error = %v", err)
	}

	err := publishTestArtifacts(store, ArtifactPublicationOptions{
		Producer: "cost",
		Results:  RawResults(map[string]string{"ok": "true"}),
		BuildReport: func() (*Report, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("PublishArtifacts() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ResultFilename("cost"))); err != nil {
		t.Fatalf("result file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ReportFilename("cost"))); !os.IsNotExist(err) {
		t.Fatalf("report file exists after nil report: %v", err)
	}
}

func TestPublishArtifactsDeletesStaleReportOnBuildError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewFileReportStore(dir)
	if err := publishTestArtifacts(store, ArtifactPublicationOptions{
		Producer: "cost",
		Results:  NoResults(),
		BuildReport: func() (*Report, error) {
			return testPublishReport(t, "cost", ArtifactContext{}), nil
		},
	}); err != nil {
		t.Fatalf("seed report error = %v", err)
	}

	wantErr := errors.New("boom")
	err := publishTestArtifacts(store, ArtifactPublicationOptions{
		Producer: "cost",
		Results:  RawResults(map[string]string{"ok": "true"}),
		BuildReport: func() (*Report, error) {
			return nil, wantErr
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("PublishArtifacts() error = %v, want %v", err, wantErr)
	}
	if _, err := os.Stat(filepath.Join(dir, ResultFilename("cost"))); err != nil {
		t.Fatalf("result file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ReportFilename("cost"))); !os.IsNotExist(err) {
		t.Fatalf("report file exists after build error: %v", err)
	}
}

func TestPublishArtifactsReportsProducerMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewFileReportStore(dir)
	err := publishTestArtifacts(store, ArtifactPublicationOptions{
		Producer: "cost",
		Results:  RawResults(map[string]string{"ok": "true"}),
		BuildReport: func() (*Report, error) {
			return testPublishReport(t, "policy", ArtifactContext{}), nil
		},
	})
	if err == nil {
		t.Fatal("PublishArtifacts() error = nil, want producer mismatch")
	}
	if !strings.Contains(err.Error(), `report producer "policy" does not match artifact producer "cost"`) {
		t.Fatalf("PublishArtifacts() error = %q, want producer mismatch", err.Error())
	}
	if _, statErr := os.Stat(filepath.Join(dir, ResultFilename("cost"))); statErr != nil {
		t.Fatalf("result file missing despite report error: %v", statErr)
	}
}

func TestPublishArtifactsJoinsBuildAndWriteErrors(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(dir, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	store := NewFileReportStore(dir)
	buildErr := errors.New("build failed")

	err := publishTestArtifacts(store, ArtifactPublicationOptions{
		Producer: "cost",
		Results:  RawResults(map[string]string{"ok": "true"}),
		BuildReport: func() (*Report, error) {
			return nil, buildErr
		},
	})
	if !errors.Is(err, buildErr) {
		t.Fatalf("PublishArtifacts() error = %v, want build error", err)
	}
	if !strings.Contains(err.Error(), "replace artifacts") {
		t.Fatalf("PublishArtifacts() error = %q, want write error context", err.Error())
	}
}

func TestPublishArtifactsNoopsWithNilStore(t *testing.T) {
	t.Parallel()

	if err := publishTestArtifacts(nil, ArtifactPublicationOptions{Producer: "cost"}); err != nil {
		t.Fatalf("PublishArtifacts(nil store) error = %v", err)
	}
}

func publishTestArtifacts(store ArtifactPublisher, opts ArtifactPublicationOptions) error {
	publication, err := NewArtifactPublication(opts)
	if err != nil {
		return err
	}
	if store == nil {
		return publishToStore(context.Background(), publication, nil)
	}
	return store.PublishArtifacts(context.Background(), publication)
}

func testPublishReport(t *testing.T, producer string, artifact ArtifactContext) *Report {
	t.Helper()

	report, err := NewRenderedReport(RenderedReportOptions{
		Producer: producer,
		Title:    "Report",
		Status:   ReportStatusPass,
		Artifact: artifact,
		Sections: []RenderedSectionOptions{{
			Title:  "Report",
			Blocks: []RenderBlock{NewTextBlock(RenderText("ok"))},
		}},
	})
	if err != nil {
		t.Fatalf("NewRenderedReport() error = %v", err)
	}
	return report
}
