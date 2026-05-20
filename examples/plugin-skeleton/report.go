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
//  1. Convert your domain result into ci.RenderBlock values. Producer plugins
//     own their analysis model; reports expose only render-ready JSON.
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
				ci.RenderTableBlock("", []string{"Field", "Value"}, [][]string{
					{"Greeting", result.Greeting},
					{"Work dir", result.WorkDir},
					{"Service dir", result.ServiceDir},
				}),
			},
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("build report: %w", err)
	}

	return report, nil
}
