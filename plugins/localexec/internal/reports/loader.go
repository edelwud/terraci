package reports

import (
	"context"
	"fmt"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/planresults"
)

const SummaryReportProducer = "summary"

type Loader interface {
	Load(ctx context.Context) (*ci.Report, error)
}

type storeLoader struct {
	reader   ci.ReportReader
	workDir  string
	segments []string
}

func NewLoader(reader ci.ReportReader, workDir string, segments []string) Loader {
	if reader == nil {
		reader = ci.NewMemoryReportStore()
	}
	return storeLoader{
		reader:   reader,
		workDir:  workDir,
		segments: append([]string(nil), segments...),
	}
}

func (l storeLoader) Load(ctx context.Context) (*ci.Report, error) {
	collection, err := planresults.Scan(l.workDir, l.segments)
	if err != nil {
		return nil, fmt.Errorf("scan plan results: %w", err)
	}
	loaded, err := l.reader.LoadReports(ctx)
	if err != nil {
		return nil, err
	}

	selection := ci.SelectCurrentReports(collection, loaded, ci.ReportSelectionOptions{
		Consumer:         "local-exec summary",
		ExcludeProducers: []string{SummaryReportProducer},
	})
	logWarnings(selection.Warnings)
	if len(selection.Reports) == 0 {
		return nil, nil
	}
	return BuildSummaryReport(collection, selection.Reports)
}

func BuildSummaryReport(collection *ci.PlanResultCollection, reports []*ci.Report) (*ci.Report, error) {
	sections := make([]ci.RenderedSectionOptions, 0)
	status := ci.ReportStatusPass
	for _, report := range reports {
		status = strictestReportStatus(status, report.Status)
		for i, section := range report.Sections {
			rendered, err := ci.DecodeRenderSection(section)
			if err != nil {
				return nil, fmt.Errorf("decode report %q section %d: %w", report.Producer, i, err)
			}
			sectionTitle := report.Title
			if section.Title() != "" && section.Title() != report.Title {
				sectionTitle = fmt.Sprintf("%s: %s", report.Title, section.Title())
			}
			sectionSummary := section.Summary()
			if sectionSummary == "" {
				sectionSummary = report.Summary
			}
			sections = append(sections, ci.RenderedSectionOptions{
				Title:   sectionTitle,
				Summary: sectionSummary,
				Status:  section.Status(),
				Blocks:  rendered.Blocks(),
			})
		}
	}
	if len(sections) == 0 {
		return nil, nil
	}

	run, err := ci.NewArtifactRun(ci.ArtifactRunOptions{
		Producer:    SummaryReportProducer,
		PlanResults: collection,
	})
	if err != nil {
		return nil, fmt.Errorf("build summary artifact run: %w", err)
	}
	return ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: SummaryReportProducer,
		Title:    "Plugin Reports",
		Status:   status,
		Summary:  fmt.Sprintf("%d plugin reports", len(reports)),
		Artifact: run.Artifact,
		Sections: sections,
	})
}

func strictestReportStatus(left, right ci.ReportStatus) ci.ReportStatus {
	if left == ci.ReportStatusFail || right == ci.ReportStatusFail {
		return ci.ReportStatusFail
	}
	if left == ci.ReportStatusWarn || right == ci.ReportStatusWarn {
		return ci.ReportStatusWarn
	}
	if left.Valid() {
		return left
	}
	if right.Valid() {
		return right
	}
	return ci.ReportStatusPass
}

func logWarnings(warnings []string) {
	for _, warning := range warnings {
		log.Warn(warning)
	}
}
