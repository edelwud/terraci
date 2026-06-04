package ci_test

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
)

func ExampleNewRenderedReport() {
	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: "policy",
		Title:    "Policy Check",
		Status:   ci.ReportStatusWarn,
		Summary:  "1 warning",
		Artifact: ci.NewArtifactContext(ci.ArtifactContextOptions{
			CommitSHA:              "abc123",
			PlanResultsFingerprint: "fingerprint",
		}),
		Sections: []ci.RenderedSectionOptions{{
			Title:   "Findings",
			Summary: "1 warning",
			Blocks: []ci.RenderBlock{
				ci.NewTableBlock("Warnings", []ci.RenderColumn{
					ci.NewRenderColumn("Module"),
					ci.NewRenderColumn("Message"),
				}, []ci.RenderRow{
					ci.NewRenderRow(ci.RenderModulePath("svc/prod/vpc"), ci.RenderText("tag missing")),
				}),
			},
		}},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(report.Producer(), report.Sections()[0].Kind())
	// Output: policy rendered
}
