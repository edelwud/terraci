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

// ArtifactPublicationOptions describes one canonical producer artifact write.
type ArtifactPublicationOptions struct {
	Producer    string
	Writer      ArtifactWriter
	Results     any
	BuildReport ArtifactReportBuilder
}

// ArtifactPublication is a validated producer artifact publication intent.
type ArtifactPublication struct {
	producer    string
	writer      ArtifactWriter
	results     any
	buildReport ArtifactReportBuilder
	constructed bool
}

// NewArtifactPublication validates and returns a publication intent.
func NewArtifactPublication(opts ArtifactPublicationOptions) (ArtifactPublication, error) {
	if err := validateArtifactProducer(opts.Producer); err != nil {
		return ArtifactPublication{}, err
	}
	return ArtifactPublication{
		producer:    opts.Producer,
		writer:      opts.Writer,
		results:     opts.Results,
		buildReport: opts.BuildReport,
		constructed: true,
	}, nil
}

// PublishArtifacts persists raw producer results and replaces the matching
// render-ready report. Raw results are always passed to the writer. If report
// construction fails or returns nil, a nil report is written so stale report
// artifacts for the producer are removed.
func PublishArtifacts(ctx context.Context, publication ArtifactPublication) error {
	if !publication.constructed {
		return errors.New("artifact publication must be built with NewArtifactPublication")
	}
	if publication.writer == nil {
		return nil
	}

	var errs []error
	var report *Report
	if publication.buildReport != nil {
		built, err := publication.buildReport()
		if err != nil {
			errs = append(errs, fmt.Errorf("build report: %w", err))
		} else {
			report = built
		}
	}

	if err := publication.writer.ReplaceResultsAndReport(ctx, publication.producer, publication.results, report); err != nil {
		errs = append(errs, fmt.Errorf("replace artifacts: %w", err))
	}
	return errors.Join(errs...)
}
