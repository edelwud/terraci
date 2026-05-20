package ci

import (
	"context"
	"errors"
	"fmt"
)

// ArtifactReportBuilder builds a render-ready report for a producer artifact
// write. Returning nil without an error intentionally removes any stale report
// while still preserving raw results.
type ArtifactReportBuilder func() (*Report, error)

// PublishArtifactsRequest describes one canonical producer artifact write.
type PublishArtifactsRequest struct {
	Producer    string
	Writer      ArtifactWriter
	Results     any
	BuildReport ArtifactReportBuilder
}

// PublishArtifacts persists raw producer results and replaces the matching
// render-ready report. Raw results are always passed to the writer. If report
// construction fails or returns nil, a nil report is written so stale report
// artifacts for the producer are removed.
func PublishArtifacts(ctx context.Context, req PublishArtifactsRequest) error {
	if req.Writer == nil {
		return nil
	}

	var errs []error
	var report *Report
	if req.BuildReport != nil {
		built, err := req.BuildReport()
		if err != nil {
			errs = append(errs, fmt.Errorf("build report: %w", err))
		} else {
			report = built
		}
	}

	if err := req.Writer.ReplaceResultsAndReport(ctx, req.Producer, req.Results, report); err != nil {
		errs = append(errs, fmt.Errorf("replace artifacts: %w", err))
	}
	return errors.Join(errs...)
}
