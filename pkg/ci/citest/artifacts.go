package citest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

const publishArtifactsResultCaseKey = "case"

// StaticReportLoader is a small ci.ReportLoader fake for report consumer tests.
type StaticReportLoader struct {
	Reports ci.ReportCollection
	Err     error
}

// NewStaticReportLoader returns a deterministic report loader fake.
func NewStaticReportLoader(reports ...*ci.Report) StaticReportLoader {
	return StaticReportLoader{Reports: ci.NewReportCollection(reports...)}
}

// LoadReports implements ci.ReportLoader.
func (l StaticReportLoader) LoadReports(context.Context) (ci.ReportCollection, error) {
	if l.Err != nil {
		return ci.ReportCollection{}, l.Err
	}
	return ci.NewReportCollection(l.Reports.Reports()...), nil
}

// PublishReport publishes a report-only artifact through the public publisher port.
func PublishReport(tb testing.TB, publisher ci.ArtifactPublisher, report *ci.Report) {
	tb.Helper()
	if publisher == nil {
		tb.Fatal("artifact publisher is nil")
	}
	if report == nil {
		tb.Fatal("report is nil")
	}
	publication, err := ci.NewArtifactPublication(ci.ArtifactPublicationOptions{
		Producer: report.Producer(),
		Results:  ci.NoResults(),
		BuildReport: func() (*ci.Report, error) {
			return report, nil
		},
	})
	if err != nil {
		tb.Fatalf("NewArtifactPublication() error = %v", err)
	}
	if err := publisher.PublishArtifacts(context.Background(), publication); err != nil {
		tb.Fatalf("PublishArtifacts(%s) error = %v", report.Producer(), err)
	}
}

// AssertPublishArtifactsContract verifies the high-level artifact publisher
// lifecycle through public ports: raw result publication succeeds, successful
// reports are visible through the loader, nil/build-error reports remove stale
// report state, and build/write errors are joined.
func AssertPublishArtifactsContract(tb testing.TB, producer string, report *ci.Report) {
	tb.Helper()
	if producer == "" {
		tb.Fatal("producer is empty")
	}
	if report == nil {
		tb.Fatal("report is nil")
	}

	successStore := ci.NewMemoryReportStore()
	successPublication := mustArtifactPublication(tb, ci.ArtifactPublicationOptions{
		Producer: producer,
		Results:  ci.RawResults(map[string]string{publishArtifactsResultCaseKey: "success"}),
		BuildReport: func() (*ci.Report, error) {
			return report, nil
		},
	})
	if err := successStore.PublishArtifacts(context.Background(), successPublication); err != nil {
		tb.Fatalf("PublishArtifacts(success) error = %v", err)
	}
	assertReportPresent(tb, successStore, producer)

	nilStore := ci.NewMemoryReportStore()
	PublishReport(tb, nilStore, report)
	nilPublication := mustArtifactPublication(tb, ci.ArtifactPublicationOptions{
		Producer: producer,
		Results:  ci.RawResults(map[string]string{publishArtifactsResultCaseKey: "nil-report"}),
		BuildReport: func() (*ci.Report, error) {
			return nil, nil
		},
	})
	if err := nilStore.PublishArtifacts(context.Background(), nilPublication); err != nil {
		tb.Fatalf("PublishArtifacts(nil report) error = %v", err)
	}
	assertReportMissing(tb, nilStore, producer)

	buildErr := errors.New("build failed")
	buildErrorStore := ci.NewMemoryReportStore()
	PublishReport(tb, buildErrorStore, report)
	buildErrorPublication := mustArtifactPublication(tb, ci.ArtifactPublicationOptions{
		Producer: producer,
		Results:  ci.RawResults(map[string]string{publishArtifactsResultCaseKey: "build-error"}),
		BuildReport: func() (*ci.Report, error) {
			return nil, buildErr
		},
	})
	err := buildErrorStore.PublishArtifacts(context.Background(), buildErrorPublication)
	if !errors.Is(err, buildErr) {
		tb.Fatalf("PublishArtifacts(build error) error = %v, want %v", err, buildErr)
	}
	assertReportMissing(tb, buildErrorStore, producer)

	blockedDir := filepath.Join(tb.TempDir(), "blocked")
	if writeErr := os.WriteFile(blockedDir, []byte("not a directory"), 0o600); writeErr != nil {
		tb.Fatalf("write blocking file: %v", writeErr)
	}
	joinedStore := ci.NewFileReportStore(blockedDir)
	joinedPublication := mustArtifactPublication(tb, ci.ArtifactPublicationOptions{
		Producer: producer,
		Results:  ci.RawResults(map[string]string{publishArtifactsResultCaseKey: "joined-errors"}),
		BuildReport: func() (*ci.Report, error) {
			return nil, buildErr
		},
	})
	err = joinedStore.PublishArtifacts(context.Background(), joinedPublication)
	if !errors.Is(err, buildErr) {
		tb.Fatalf("PublishArtifacts(joined errors) error = %v, want build error", err)
	}
}

func mustArtifactPublication(tb testing.TB, opts ci.ArtifactPublicationOptions) ci.ArtifactPublication {
	tb.Helper()
	publication, err := ci.NewArtifactPublication(opts)
	if err != nil {
		tb.Fatalf("NewArtifactPublication() error = %v", err)
	}
	return publication
}

func assertReportPresent(tb testing.TB, loader ci.ReportLoader, producer string) {
	tb.Helper()
	collection, err := loader.LoadReports(context.Background())
	if err != nil {
		tb.Fatalf("LoadReports() error = %v", err)
	}
	if _, ok := collection.Find(producer); !ok {
		tb.Fatalf("Find(%s) ok = false, want true", producer)
	}
}

func assertReportMissing(tb testing.TB, loader ci.ReportLoader, producer string) {
	tb.Helper()
	collection, err := loader.LoadReports(context.Background())
	if err != nil {
		tb.Fatalf("LoadReports() error = %v", err)
	}
	if _, ok := collection.Find(producer); ok {
		tb.Fatalf("Find(%s) ok = true, want false", producer)
	}
}
