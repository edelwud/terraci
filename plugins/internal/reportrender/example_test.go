package reportrender_test

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/plugins/internal/reportrender"
)

func ExampleMarkdownReport() {
	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: "policy",
		Title:    "Policy Check",
		Status:   ci.ReportStatusWarn,
		Summary:  "1 warning",
	})
	if err != nil {
		panic(err)
	}

	rendered, err := reportrender.MarkdownReport(report)
	if err != nil {
		panic(err)
	}

	fmt.Println(strings.TrimSpace(rendered))
	// Output:
	// ### Warning Policy Check
	//
	// 1 warning
}
