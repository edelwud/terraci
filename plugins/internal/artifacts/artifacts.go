package artifacts

import (
	"context"
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
)

type ReportBuilder func(ci.ArtifactRun) (*ci.Report, error)

type ReplaceRequest struct {
	Producer    string
	Writer      ci.ArtifactWriter
	Results     any
	Run         ci.ArtifactRun
	RunError    error
	BuildReport ReportBuilder
}

// ReplaceResultsAndReport persists producer raw results and the optional
// render-ready report. Raw results are always handed to the writer; when report
// construction fails or is skipped, the writer receives a nil report so stale
// report artifacts are removed.
func ReplaceResultsAndReport(ctx context.Context, req ReplaceRequest) error {
	if req.Writer == nil {
		return nil
	}

	var errs []error
	var report *ci.Report
	if req.RunError != nil {
		errs = append(errs, fmt.Errorf("artifact run: %w", req.RunError))
	} else if req.BuildReport != nil {
		built, err := req.BuildReport(req.Run)
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
