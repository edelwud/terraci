package skeleton

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
)

// --- Producer pattern -------------------------------------------------------
//
// Built-in references: plugins/cost/report.go, plugins/policy/report.go,
// plugins/tfupdate/report.go.
//
// Steps:
//
//  1. Convert your domain result into ci.RenderBlock values. Producer plugins
//     own their analysis model; reports expose only render-ready JSON.
//
//  2. Build a ci.ReportSection via ci.EncodeRenderSection.
//
//  3. Compose the final ci.Report and persist it via ci.SaveResultsAndReport
//     — that helper also handles the "save results JSON alongside" case if
//     you have raw analysis output.
//
//  4. Always populate Provenance via ci.NewProvenance(). Local consumers
//     (localexec/render) compare the fingerprint against the live workspace
//     to decide whether the on-disk report is still trustworthy.

func runProducer(_ context.Context, appCtx *plugin.AppContext, cfg *Config) error {
	section, err := ci.EncodeRenderSection(
		"Skeleton payload",
		"one demo section",
		ci.ReportStatusPass,
		ci.RenderTableBlock("", []string{"Field", "Value"}, [][]string{
			{"Greeting", cfg.Greeting},
			{"Work dir", appCtx.WorkDir()},
			{"Service dir", appCtx.ServiceDir()},
		}),
	)
	if err != nil {
		return fmt.Errorf("encode section: %w", err)
	}

	report := &ci.Report{
		Producer: pluginName,
		Title:    "Skeleton Report",
		Status:   ci.ReportStatusPass,
		Summary:  "skeleton payload generated",
		// CommitSHA / PipelineID can be sourced from a CI provider when
		// available — see plugins/summary for the canonical pattern.
		Provenance: ci.NewProvenance("", "", ""),
		Sections:   []ci.ReportSection{section},
	}

	// SaveResultsAndReport handles directory creation and atomic JSON write.
	// Pass nil for the results filename if you don't have a separate raw
	// payload to persist — only the report goes to disk.
	if err := ci.SaveResultsAndReport(appCtx.ServiceDir(), "", nil, report); err != nil {
		return fmt.Errorf("save report: %w", err)
	}

	fmt.Printf("%s\n", cfg.Greeting)
	fmt.Printf("wrote %s/%s\n", appCtx.ServiceDir(), ci.ReportFilename(pluginName))
	return nil
}
