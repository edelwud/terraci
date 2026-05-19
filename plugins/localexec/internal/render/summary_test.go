package render

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
)

func TestRenderSummaryReportCLI_RendersStructuredSections(t *testing.T) {
	t.Parallel()

	report := &ci.Report{
		Producer: summaryReportProducer,
		Title:    "Terraform Plan Summary",
		Summary:  "2 modules: 1 with changes, 1 no changes, 0 failed",
		Sections: []ci.ReportSection{
			citest.MustRenderedSection(
				"Summary",
				"2 modules: 1 with changes, 1 no changes, 0 failed",
				ci.ReportStatusWarn,
				ci.RenderListBlock("", []string{"warn Cost Estimation: 1 module added cost"}),
			),
			citest.MustRenderedSection(
				"Environment: `prod`",
				"1 actionable modules",
				ci.ReportStatusWarn,
				ci.RenderTableBlock("", []string{"Status", "Module", "Summary"}, [][]string{{"changes", "svc/prod/eu/vpc", "+1"}}),
				ci.RenderDetailsBlock("svc/prod/eu/vpc (+1)", "### Resources\n- aws_vpc.main (create)\n\n#### Full plan output\n\n```diff\n+ resource \"aws_vpc\" \"main\"\n```", ""),
			),
		},
	}

	rendered, err := SummaryReportCLI(report)
	if err != nil {
		t.Fatalf("SummaryReportCLI() error = %v", err)
	}
	for _, wanted := range []string{
		"Terraform Plan Summary",
		"2 modules: 1 with changes, 1 no changes, 0 failed",
		"Summary",
		"• warn Cost Estimation: 1 module added cost",
		"Environment: `prod`",
		"svc/prod/eu/vpc (+1)",
		"Resources",
		`+ resource "aws_vpc" "main"`,
		"┌",
	} {
		if !strings.Contains(rendered, wanted) {
			t.Fatalf("rendered output missing %q:\n%s", wanted, rendered)
		}
	}
}
