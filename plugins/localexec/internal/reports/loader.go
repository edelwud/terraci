package reports

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/planresults"
	"github.com/edelwud/terraci/plugins/internal/diagnosticlog"
)

const SummaryReportProducer = "summary"

type Loader interface {
	Load(ctx context.Context) (*Result, error)
}

type Result struct {
	report      *ci.Report
	diagnostics diagnostic.List
}

func NewResult(report *ci.Report, diagnostics diagnostic.List) *Result {
	return &Result{report: report.Clone(), diagnostics: diagnostics}
}

func (r *Result) Report() *ci.Report {
	if r == nil {
		return nil
	}
	return r.report.Clone()
}

func (r *Result) Diagnostics() diagnostic.List {
	if r == nil {
		return diagnostic.List{}
	}
	return r.diagnostics
}

type storeLoader struct {
	reader   ci.ReportLoader
	workDir  string
	segments []string
}

func NewLoader(reader ci.ReportLoader, workDir string, segments []string) Loader {
	if reader == nil {
		reader = ci.NewMemoryReportStore()
	}
	return storeLoader{
		reader:   reader,
		workDir:  workDir,
		segments: append([]string(nil), segments...),
	}
}

func (l storeLoader) Load(ctx context.Context) (*Result, error) {
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
	diagnostics := selection.Diagnostics()
	diagnosticlog.Log(diagnostics)
	reports := selection.Reports()
	if len(reports) == 0 {
		return NewResult(nil, diagnostics), nil
	}
	report, err := BuildSummaryReport(collection, reports)
	if err != nil {
		return nil, err
	}
	return NewResult(report, diagnostics), nil
}

func BuildSummaryReport(collection *ci.PlanResultCollection, reports []*ci.Report) (*ci.Report, error) {
	sections := make([]ci.RenderedSectionOptions, 0)
	status := ci.ReportStatusPass
	for _, report := range reports {
		status = strictestReportStatus(status, report.Status())
		for i, section := range report.Sections() {
			rendered, err := ci.DecodeRenderSection(section)
			if err != nil {
				return nil, fmt.Errorf("decode report %q section %d: %w", report.Producer(), i, err)
			}
			reportTitle := report.Title()
			sectionTitle := reportTitle
			if section.Title() != "" && section.Title() != reportTitle {
				sectionTitle = fmt.Sprintf("%s: %s", reportTitle, section.Title())
			}
			sectionSummary := section.Summary()
			if sectionSummary == "" {
				sectionSummary = report.Summary()
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
		Artifact: run.Artifact(),
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
