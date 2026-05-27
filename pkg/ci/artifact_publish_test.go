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

	err := PublishArtifacts(context.Background(), PublishArtifactsRequest{
		Producer: "cost",
		Writer:   store,
		Results:  map[string]string{"ok": "true"},
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
	if err := store.SaveReport(context.Background(), testPublishReport(t, "cost", ArtifactContext{})); err != nil {
		t.Fatalf("SaveReport() error = %v", err)
	}

	err := PublishArtifacts(context.Background(), PublishArtifactsRequest{
		Producer: "cost",
		Writer:   store,
		Results:  map[string]string{"ok": "true"},
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
	if err := store.SaveReport(context.Background(), testPublishReport(t, "cost", ArtifactContext{})); err != nil {
		t.Fatalf("SaveReport() error = %v", err)
	}

	wantErr := errors.New("boom")
	err := PublishArtifacts(context.Background(), PublishArtifactsRequest{
		Producer: "cost",
		Writer:   store,
		Results:  map[string]string{"ok": "true"},
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
	err := PublishArtifacts(context.Background(), PublishArtifactsRequest{
		Producer: "cost",
		Writer:   store,
		Results:  map[string]string{"ok": "true"},
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

func TestPublishArtifactsJoinsBuildAndWriterErrors(t *testing.T) {
	t.Parallel()

	buildErr := errors.New("build failed")
	writerErr := errors.New("writer failed")
	err := PublishArtifacts(context.Background(), PublishArtifactsRequest{
		Producer: "cost",
		Writer:   fakePublishArtifactWriter{err: writerErr},
		Results:  map[string]string{"ok": "true"},
		BuildReport: func() (*Report, error) {
			return nil, buildErr
		},
	})
	if !errors.Is(err, buildErr) || !errors.Is(err, writerErr) {
		t.Fatalf("PublishArtifacts() error = %v, want joined build and writer errors", err)
	}
}

func TestPublishArtifactsNoopsWithoutWriter(t *testing.T) {
	t.Parallel()

	if err := PublishArtifacts(context.Background(), PublishArtifactsRequest{Producer: "cost"}); err != nil {
		t.Fatalf("PublishArtifacts(nil writer) error = %v", err)
	}
}

type fakePublishArtifactWriter struct {
	err error
}

func (w fakePublishArtifactWriter) SaveResults(context.Context, string, any) error {
	return w.err
}

func (w fakePublishArtifactWriter) ReplaceResultsAndReport(context.Context, string, any, *Report) error {
	return w.err
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
