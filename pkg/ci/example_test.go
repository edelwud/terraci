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
		Sections: []ci.RenderedSectionOptions{{
			Title:   "Findings",
			Summary: "1 warning",
			Blocks: []ci.RenderBlock{
				ci.RenderTableBlock("Warnings", []string{"Module", "Message"}, [][]string{{"svc/prod/vpc", "tag missing"}}),
			},
		}},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(report.Producer, report.Sections[0].Kind())
	// Output: policy rendered
}
