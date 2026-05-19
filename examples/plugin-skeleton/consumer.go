package skeleton

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
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
//  2. Branch on report.Producer when needed. Decode render-ready sections via
//     ci.DecodeRenderSection; external plugins should not parse payload JSON
//     by hand.
//
//  3. When your consumer has the current PlanResultCollection, pass it to
//     ci.SelectCurrentReports so stale reports are skipped consistently.

func runConsumer(ctx context.Context, appCtx *plugin.AppContext) error {
	reports, err := appCtx.Reports().LoadReports(ctx)
	if err != nil {
		return fmt.Errorf("load reports: %w", err)
	}

	selection := ci.SelectCurrentReports(nil, reports, ci.ReportSelectionOptions{
		Consumer:         pluginName,
		ExcludeProducers: []string{pluginName},
	})
	if len(selection.Reports) == 0 {
		fmt.Println("no reports found in service directory")
		return nil
	}

	for _, r := range selection.Reports {
		fmt.Printf("- %s [%s] %s\n", r.Producer, r.Status, r.Summary)

		// Optional: decode render-ready section payloads.
		for _, section := range r.Sections {
			rendered, err := ci.DecodeRenderSection(section)
			if err != nil {
				fmt.Printf("    %s: decode error: %v\n", section.Title(), err)
				continue
			}
			fmt.Printf("    %s: %d block(s)\n", section.Title(), len(rendered.Blocks))
		}
	}

	return nil
}
