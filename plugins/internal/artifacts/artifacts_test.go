package artifacts

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestReplaceResultsAndReportWritesResultsAndReport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := ci.NewFileReportStore(dir)
	run := testArtifactRun(t)

	err := ReplaceResultsAndReport(context.Background(), ReplaceRequest{
		Producer: "cost",
		Writer:   store,
		Results:  map[string]string{"ok": "true"},
		Run:      run,
		BuildReport: func(run ci.ArtifactRun) (*ci.Report, error) {
			return testReport(t, run.Producer, run.Artifact), nil
		},
	})
	if err != nil {
		t.Fatalf("ReplaceResultsAndReport() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ci.ResultFilename("cost"))); err != nil {
		t.Fatalf("result file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ci.ReportFilename("cost"))); err != nil {
		t.Fatalf("report file missing: %v", err)
	}
}

func TestReplaceResultsAndReportDeletesStaleReportOnBuildError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := ci.NewFileReportStore(dir)
	if err := store.SaveReport(context.Background(), testReport(t, "cost", ci.ArtifactContext{})); err != nil {
		t.Fatalf("SaveReport() error = %v", err)
	}

	wantErr := errors.New("boom")
	err := ReplaceResultsAndReport(context.Background(), ReplaceRequest{
		Producer: "cost",
		Writer:   store,
		Results:  map[string]string{"ok": "true"},
		Run:      testArtifactRun(t),
		BuildReport: func(ci.ArtifactRun) (*ci.Report, error) {
			return nil, wantErr
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("ReplaceResultsAndReport() error = %v, want %v", err, wantErr)
	}
	if _, err := os.Stat(filepath.Join(dir, ci.ResultFilename("cost"))); err != nil {
		t.Fatalf("result file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ci.ReportFilename("cost"))); !os.IsNotExist(err) {
		t.Fatalf("report file exists after build error: %v", err)
	}
}

func TestReplaceResultsAndReportReportsProducerMismatch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := ci.NewFileReportStore(dir)
	err := ReplaceResultsAndReport(context.Background(), ReplaceRequest{
		Producer: "cost",
		Writer:   store,
		Results:  map[string]string{"ok": "true"},
		Run:      testArtifactRun(t),
		BuildReport: func(run ci.ArtifactRun) (*ci.Report, error) {
			return testReport(t, "policy", run.Artifact), nil
		},
	})
	if err == nil {
		t.Fatal("ReplaceResultsAndReport() error = nil, want producer mismatch")
	}
	if !strings.Contains(err.Error(), `report producer "policy" does not match artifact producer "cost"`) {
		t.Fatalf("ReplaceResultsAndReport() error = %q, want producer mismatch", err.Error())
	}
	if _, statErr := os.Stat(filepath.Join(dir, ci.ResultFilename("cost"))); statErr != nil {
		t.Fatalf("result file missing despite report error: %v", statErr)
	}
}

func TestReplaceResultsAndReportJoinsBuildAndWriterErrors(t *testing.T) {
	t.Parallel()

	buildErr := errors.New("build failed")
	writerErr := errors.New("writer failed")
	err := ReplaceResultsAndReport(context.Background(), ReplaceRequest{
		Producer: "cost",
		Writer:   fakeArtifactWriter{err: writerErr},
		Results:  map[string]string{"ok": "true"},
		Run:      testArtifactRun(t),
		BuildReport: func(ci.ArtifactRun) (*ci.Report, error) {
			return nil, buildErr
		},
	})
	if !errors.Is(err, buildErr) || !errors.Is(err, writerErr) {
		t.Fatalf("ReplaceResultsAndReport() error = %v, want joined build and writer errors", err)
	}
}

func TestReplaceResultsAndReportNoopsWithoutWriter(t *testing.T) {
	t.Parallel()

	if err := ReplaceResultsAndReport(context.Background(), ReplaceRequest{Producer: "cost"}); err != nil {
		t.Fatalf("ReplaceResultsAndReport(nil writer) error = %v", err)
	}
}

type fakeArtifactWriter struct {
	err error
}

func (w fakeArtifactWriter) SaveResults(context.Context, string, any) error {
	return w.err
}

func (w fakeArtifactWriter) ReplaceResultsAndReport(context.Context, string, any, *ci.Report) error {
	return w.err
}

func testArtifactRun(t *testing.T) ci.ArtifactRun {
	t.Helper()

	run, err := ci.NewArtifactRun(ci.ArtifactRunOptions{Producer: "cost"})
	if err != nil {
		t.Fatalf("NewArtifactRun() error = %v", err)
	}
	return run
}

func testReport(t *testing.T, producer string, artifact ci.ArtifactContext) *ci.Report {
	t.Helper()

	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: producer,
		Title:    "Report",
		Status:   ci.ReportStatusPass,
		Artifact: artifact,
		Sections: []ci.RenderedSectionOptions{{
			Title:  "Report",
			Blocks: []ci.RenderBlock{ci.RenderTextBlock("ok")},
		}},
	})
	if err != nil {
		t.Fatalf("NewRenderedReport() error = %v", err)
	}
	return report
}
