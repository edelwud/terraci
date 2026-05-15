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
// internal/render/report_loader.go.
//
// Steps:
//
//  1. ci.LoadReports(serviceDir) returns every *-report.json in
//     deterministic order. Filter out your own producer to avoid an
//     accidental self-loop.
//
//  2. Branch on report.Producer or section.Kind to decide what to render.
//     Decode opaque payloads via ci.DecodeSection[T] when you actually
//     need the typed data — falling back to the section's pre-rendered
//     Title/SectionSummary is fine for at-a-glance views.
//
//  3. Validate report.Provenance against the live workspace if your
//     consumer's correctness depends on the report being current. The
//     summary plugin compares plan_results_fingerprint to the in-memory
//     PlanResultCollection; localexec uses the same idea before
//     re-rendering a stale comment locally.

func runConsumer(_ context.Context, appCtx *plugin.AppContext) error {
	reports, err := ci.LoadReports(appCtx.ServiceDir())
	if err != nil {
		return fmt.Errorf("load reports: %w", err)
	}

	if len(reports) == 0 {
		fmt.Println("no reports found in service directory")
		return nil
	}

	for _, r := range reports {
		// Skip our own producer output — consumers don't echo themselves.
		if r.Producer == pluginName {
			continue
		}
		fmt.Printf("- %s [%s] %s\n", r.Producer, r.Status, r.Summary)

		// Optional: decode render-ready section payloads.
		for _, section := range r.Sections {
			if section.Kind == ci.ReportSectionKindRendered {
				rendered, err := ci.DecodeSection[ci.RenderSection](section)
				if err != nil {
					fmt.Printf("    rendered: decode error: %v\n", err)
					continue
				}
				fmt.Printf("    %s: %d block(s)\n", section.Title, len(rendered.Blocks))
				continue
			}

			// Opaque payload — display only the producer-supplied title.
			fmt.Printf("    %s: %s\n", section.Kind, section.Title)
		}
	}

	return nil
}
