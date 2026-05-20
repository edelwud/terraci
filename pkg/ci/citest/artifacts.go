package citest

import (
	"context"
	"errors"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

const publishArtifactsResultCaseKey = "case"

// RecordedResultWrite captures one SaveResults call.
type RecordedResultWrite struct {
	Producer string
	Results  any
}

// RecordedArtifactReplace captures one ReplaceResultsAndReport call.
type RecordedArtifactReplace struct {
	Producer string
	Results  any
	Report   *ci.Report
}

// RecordingArtifactWriter is a small ci.ArtifactWriter fake for producer
// artifact lifecycle tests.
type RecordingArtifactWriter struct {
	SaveResultsError error
	ReplaceError     error
	ResultWrites     []RecordedResultWrite
	ReplaceWrites    []RecordedArtifactReplace
}

// SaveResults implements ci.ArtifactWriter.
func (w *RecordingArtifactWriter) SaveResults(_ context.Context, producer string, results any) error {
	w.ResultWrites = append(w.ResultWrites, RecordedResultWrite{
		Producer: producer,
		Results:  results,
	})
	return w.SaveResultsError
}

// ReplaceResultsAndReport implements ci.ArtifactWriter.
func (w *RecordingArtifactWriter) ReplaceResultsAndReport(_ context.Context, producer string, results any, report *ci.Report) error {
	var cloned *ci.Report
	if report != nil {
		cloned = report.Clone()
	}
	w.ReplaceWrites = append(w.ReplaceWrites, RecordedArtifactReplace{
		Producer: producer,
		Results:  results,
		Report:   cloned,
	})
	return w.ReplaceError
}

// AssertPublishArtifactsContract verifies the high-level ci.PublishArtifacts
// lifecycle against a recording writer: raw results are always sent through
// ReplaceResultsAndReport, successful reports are saved, nil/build-error
// reports delete stale report state, errors are joined, and nil writers noop.
func AssertPublishArtifactsContract(tb testing.TB, producer string, report *ci.Report) {
	tb.Helper()
	if producer == "" {
		tb.Fatal("producer is empty")
	}
	if report == nil {
		tb.Fatal("report is nil")
	}

	successWriter := &RecordingArtifactWriter{}
	if err := ci.PublishArtifacts(context.Background(), ci.PublishArtifactsRequest{
		Producer: producer,
		Writer:   successWriter,
		Results:  map[string]string{publishArtifactsResultCaseKey: "success"},
		BuildReport: func() (*ci.Report, error) {
			return report, nil
		},
	}); err != nil {
		tb.Fatalf("PublishArtifacts(success) error = %v", err)
	}
	assertSingleReplace(tb, successWriter, producer, true)

	nilWriter := &RecordingArtifactWriter{}
	if err := ci.PublishArtifacts(context.Background(), ci.PublishArtifactsRequest{
		Producer: producer,
		Writer:   nilWriter,
		Results:  map[string]string{publishArtifactsResultCaseKey: "nil-report"},
		BuildReport: func() (*ci.Report, error) {
			return nil, nil
		},
	}); err != nil {
		tb.Fatalf("PublishArtifacts(nil report) error = %v", err)
	}
	assertSingleReplace(tb, nilWriter, producer, false)

	buildErr := errors.New("build failed")
	buildErrorWriter := &RecordingArtifactWriter{}
	err := ci.PublishArtifacts(context.Background(), ci.PublishArtifactsRequest{
		Producer: producer,
		Writer:   buildErrorWriter,
		Results:  map[string]string{publishArtifactsResultCaseKey: "build-error"},
		BuildReport: func() (*ci.Report, error) {
			return nil, buildErr
		},
	})
	if !errors.Is(err, buildErr) {
		tb.Fatalf("PublishArtifacts(build error) error = %v, want %v", err, buildErr)
	}
	assertSingleReplace(tb, buildErrorWriter, producer, false)

	writeErr := errors.New("write failed")
	joinedWriter := &RecordingArtifactWriter{ReplaceError: writeErr}
	err = ci.PublishArtifacts(context.Background(), ci.PublishArtifactsRequest{
		Producer: producer,
		Writer:   joinedWriter,
		Results:  map[string]string{publishArtifactsResultCaseKey: "joined-errors"},
		BuildReport: func() (*ci.Report, error) {
			return nil, buildErr
		},
	})
	if !errors.Is(err, buildErr) || !errors.Is(err, writeErr) {
		tb.Fatalf("PublishArtifacts(joined errors) error = %v, want build and write errors", err)
	}
	assertSingleReplace(tb, joinedWriter, producer, false)

	if err := ci.PublishArtifacts(context.Background(), ci.PublishArtifactsRequest{
		Producer: producer,
		Results:  map[string]string{publishArtifactsResultCaseKey: "nil-writer"},
	}); err != nil {
		tb.Fatalf("PublishArtifacts(nil writer) error = %v", err)
	}
}

func assertSingleReplace(tb testing.TB, writer *RecordingArtifactWriter, producer string, wantReport bool) {
	tb.Helper()
	if len(writer.ReplaceWrites) != 1 {
		tb.Fatalf("ReplaceResultsAndReport calls = %d, want 1", len(writer.ReplaceWrites))
	}
	got := writer.ReplaceWrites[0]
	if got.Producer != producer {
		tb.Fatalf("ReplaceResultsAndReport producer = %q, want %q", got.Producer, producer)
	}
	if got.Results == nil {
		tb.Fatal("ReplaceResultsAndReport results = nil")
	}
	if wantReport && got.Report == nil {
		tb.Fatal("ReplaceResultsAndReport report = nil, want report")
	}
	if !wantReport && got.Report != nil {
		tb.Fatalf("ReplaceResultsAndReport report = %#v, want nil", got.Report)
	}
}
