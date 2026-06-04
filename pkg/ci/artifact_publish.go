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

// ArtifactResults describes whether a producer publication should write raw
// results and, if so, what JSON-serializable value should be written.
type ArtifactResults struct {
	value any
	write bool
}

// RawResults marks value for persistence as {producer}-results.json.
func RawResults(value any) ArtifactResults {
	return ArtifactResults{value: value, write: true}
}

// NoResults marks a report-only publication.
func NoResults() ArtifactResults {
	return ArtifactResults{}
}

func (r ArtifactResults) valueToWrite() (any, bool) {
	return r.value, r.write
}

// ArtifactPublicationOptions describes one canonical producer artifact write.
type ArtifactPublicationOptions struct {
	Producer    string
	Results     ArtifactResults
	BuildReport ArtifactReportBuilder
}

// ArtifactPublication is a validated producer artifact publication intent.
type ArtifactPublication struct {
	producer    string
	results     ArtifactResults
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
		results:     opts.Results,
		buildReport: opts.BuildReport,
		constructed: true,
	}, nil
}

func buildPublicationReport(publication ArtifactPublication) (*Report, error) {
	if !publication.constructed {
		return nil, errors.New("artifact publication must be built with NewArtifactPublication")
	}
	if publication.buildReport != nil {
		built, err := publication.buildReport()
		if err != nil {
			return nil, fmt.Errorf("build report: %w", err)
		}
		return built, nil
	}
	return nil, nil
}

func contextPublicationError(ctx context.Context, publication ArtifactPublication) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if !publication.constructed {
		return errors.New("artifact publication must be built with NewArtifactPublication")
	}
	return nil
}
