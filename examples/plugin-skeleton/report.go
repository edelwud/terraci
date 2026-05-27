package skeleton

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
)

// --- Producer pattern -------------------------------------------------------
//
// Built-in references: plugins/cost/report.go, plugins/policy/report.go,
// plugins/tfupdate/report.go.
//
// Steps:
//
//  1. Convert your domain result into constructor-built ci.RenderBlock and
//     ci.RenderValue values. Producer plugins own their analysis model; reports
//     expose only typed render-ready JSON.
//
//  2. Compose the final ci.Report via ci.NewRenderedReport.
//
//  3. Persist raw results and the report via ci.PublishArtifacts.
//
//  4. Always pass ci.ArtifactRun.Artifact into ci.NewRenderedReport. Local
//     consumers compare the fingerprint against the live workspace to decide
//     whether the on-disk report is still trustworthy.
func buildReport(result *ProducerResult, run ci.ArtifactRun) (*ci.Report, error) {
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
				ci.NewTableBlock("", []ci.RenderColumn{
					ci.NewRenderColumn("Field"),
					ci.NewRenderColumn("Value"),
				}, []ci.RenderRow{
					ci.NewRenderRow(ci.RenderText("Greeting"), ci.RenderText(result.Greeting)),
					ci.NewRenderRow(ci.RenderText("Work dir"), ci.RenderCode(result.WorkDir)),
					ci.NewRenderRow(ci.RenderText("Service dir"), ci.RenderCode(result.ServiceDir)),
				}),
			},
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("build report: %w", err)
	}

	return report, nil
}
