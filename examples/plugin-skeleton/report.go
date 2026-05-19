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
//  2. Compose the final ci.Report via ci.NewRenderedReport.
//
//  3. Persist it via appCtx.Reports().SaveReport, or
//     ReplaceResultsAndReport when you also have raw analysis output.
//
//  4. Always pass ci.ArtifactRun.Artifact into ci.NewRenderedReport. Local
//     consumers compare the fingerprint against the live workspace to decide
//     whether the on-disk report is still trustworthy.

func runProducer(ctx context.Context, appCtx *plugin.AppContext, cfg *Config) error {
	run, err := ci.NewArtifactRun(ci.ArtifactRunOptions{
		Producer: pluginName,
		Artifact: ci.NewArtifactContext(ci.ArtifactContextOptions{
			ServiceDir: appCtx.ServiceDir(),
			WorkDir:    appCtx.WorkDir(),
		}),
	})
	if err != nil {
		return fmt.Errorf("build artifact run: %w", err)
	}

	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: pluginName,
		Title:    "Skeleton Report",
		Status:   ci.ReportStatusPass,
		Summary:  "skeleton payload generated",
		Artifact: run.Artifact,
		Sections: []ci.RenderedSectionOptions{{
			Title:   "Skeleton payload",
			Summary: "one demo section",
			Blocks: []ci.RenderBlock{
				ci.RenderTableBlock("", []string{"Field", "Value"}, [][]string{
					{"Greeting", cfg.Greeting},
					{"Work dir", appCtx.WorkDir()},
					{"Service dir", appCtx.ServiceDir()},
				}),
			},
		}},
	})
	if err != nil {
		return fmt.Errorf("build report: %w", err)
	}

	// SaveReport handles directory creation and canonical artifact filenames.
	if err := appCtx.Reports().SaveReport(ctx, report); err != nil {
		return fmt.Errorf("save report: %w", err)
	}

	fmt.Printf("%s\n", cfg.Greeting)
	fmt.Printf("wrote %s/%s\n", appCtx.ServiceDir(), ci.ReportFilename(pluginName))
	return nil
}
