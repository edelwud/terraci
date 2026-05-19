package render

import (
	"context"
	"fmt"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/planresults"
)

const summaryReportProducer = "summary"

type SummaryReportLoader interface {
	Reset(ctx context.Context) error
	Load(ctx context.Context) (*ci.Report, error)
}

type storeSummaryReportLoader struct {
	store    ci.ReportStore
	workDir  string
	segments []string
}

func NewSummaryReportLoader(store ci.ReportStore, workDir string, segments []string) SummaryReportLoader {
	if store == nil {
		store = ci.NewMemoryReportStore()
	}
	return storeSummaryReportLoader{
		store:    store,
		workDir:  workDir,
		segments: append([]string(nil), segments...),
	}
}

func (l storeSummaryReportLoader) Reset(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func (l storeSummaryReportLoader) Load(ctx context.Context) (*ci.Report, error) {
	collection, err := planresults.Scan(l.workDir, l.segments)
	if err != nil {
		return nil, fmt.Errorf("scan plan results: %w", err)
	}
	reports, err := l.store.LoadReports(ctx)
	if err != nil {
		return nil, err
	}

	selected := selectCurrentReports(collection, reports)
	if len(selected) == 0 {
		return nil, nil
	}
	return buildLocalSummaryReport(collection, selected)
}

func selectCurrentReports(collection *ci.PlanResultCollection, reports []*ci.Report) []*ci.Report {
	fingerprint := collection.Fingerprint()
	selected := make([]*ci.Report, 0, len(reports))
	for _, report := range reports {
		if report == nil || report.Producer == summaryReportProducer {
			continue
		}
		if shouldSkipStaleReport(report, fingerprint) {
			log.Warn(fmt.Sprintf("local-exec summary report %q skipped: plan_results_fingerprint %q does not match current %q",
				report.Producer, report.Provenance.PlanResultsFingerprint, fingerprint))
			continue
		}
		selected = append(selected, report)
	}
	return selected
}

func shouldSkipStaleReport(report *ci.Report, currentFingerprint string) bool {
	if report == nil || report.Provenance == nil {
		return false
	}
	reportFingerprint := report.Provenance.PlanResultsFingerprint
	return reportFingerprint != "" && currentFingerprint != "" && reportFingerprint != currentFingerprint
}

func buildLocalSummaryReport(collection *ci.PlanResultCollection, reports []*ci.Report) (*ci.Report, error) {
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
				Blocks:  rendered.Blocks,
			})
		}
	}
	if len(sections) == 0 {
		return nil, nil
	}
	return ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: summaryReportProducer,
		Title:    "Plugin Reports",
		Status:   status,
		Summary:  fmt.Sprintf("%d plugin reports", len(reports)),
		Artifact: ci.NewArtifactContext(ci.ArtifactContextOptions{
			PlanResultsFingerprint: collection.Fingerprint(),
		}),
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
