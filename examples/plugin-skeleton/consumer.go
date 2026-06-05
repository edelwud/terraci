package skeleton

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
)

// --- Consumer pattern -------------------------------------------------------
//
// Built-in references: plugins/summary/usecases.go, plugins/localexec/
// internal/reports/loader.go.
//
// Steps:
//
//  1. appCtx.Reports().LoadReports(ctx) returns every available report in
//     deterministic order. Pass those reports through ci.SelectCurrentReports;
//     exclude your own producer to avoid an accidental self-loop.
//
//  2. Branch on report.Producer() when needed. Decode render-ready sections via
//     ci.DecodeRenderSection; external plugins should not parse payload JSON
//     by hand.
//
//  3. When your consumer has the current PlanResultCollection, pass it to
//     ci.SelectCurrentReports so stale reports are skipped consistently.

func consumeReports(ctx context.Context, runtime Runtime) (*ConsumerResult, error) {
	if runtime.Reports == nil {
		return &ConsumerResult{}, nil
	}
	reports, err := runtime.Reports.LoadReports(ctx)
	if err != nil {
		return nil, fmt.Errorf("load reports: %w", err)
	}

	selection := ci.SelectCurrentReports(nil, reports, ci.ReportSelectionOptions{
		Consumer:         pluginName,
		ExcludeProducers: []string{pluginName},
	})
	selectedReports := selection.ReportCollection()
	result := &ConsumerResult{Reports: make([]ConsumedReport, 0, selectedReports.Len())}
	if selectedReports.Len() == 0 {
		return result, nil
	}

	for _, r := range selectedReports.Reports() {
		sections := r.Sections()
		consumed := ConsumedReport{
			Producer: r.Producer(),
			Status:   r.Status(),
			Summary:  r.Summary(),
			Sections: make([]ConsumedSection, 0, len(sections)),
		}
		for _, section := range sections {
			entry := ConsumedSection{Title: section.Title()}
			rendered, err := ci.DecodeRenderSection(section)
			if err != nil {
				entry.Error = err.Error()
				consumed.Sections = append(consumed.Sections, entry)
				continue
			}
			entry.Blocks = len(rendered.Blocks())
			consumed.Sections = append(consumed.Sections, entry)
		}
		result.Reports = append(result.Reports, consumed)
	}

	return result, nil
}
