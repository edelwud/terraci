package render

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestRenderSummaryReportCLI_RendersStructuredSections(t *testing.T) {
	t.Parallel()

	report := &ci.Report{
		Plugin:  "summary",
		Title:   "Terraform Plan Summary",
		Summary: "2 modules: 1 with changes, 1 no changes, 0 failed",
		Sections: []ci.ReportSection{
			{
				Kind:           ci.ReportSectionKindOverview,
				Title:          "Summary",
				Status:         ci.ReportStatusWarn,
				SectionSummary: "2 modules: 1 with changes, 1 no changes, 0 failed",
				Overview: &ci.OverviewSection{
					PlanStats: ci.SummaryPlanStats{Total: 2, Changes: 1, NoChanges: 1, Success: 2},
					Reports: []ci.SummaryReportOverview{
						{Kind: "cost_changes", Title: "Cost Estimation", Status: ci.ReportStatusWarn, Summary: "1 module added cost"},
					},
				},
			},
			{
				Kind:           ci.ReportSectionKindModuleTable,
				Title:          "Environment: `prod`",
				Status:         ci.ReportStatusWarn,
				SectionSummary: "1 actionable modules",
				ModuleTable: &ci.ModuleTableSection{
					Environment: "prod",
					Rows: []ci.ModuleTableRow{{
						ModuleID:          "svc/prod/eu/vpc",
						ModulePath:        "svc/prod/eu/vpc",
						Status:            ci.PlanStatusChanges,
						Summary:           "+1",
						StructuredDetails: "### Resources\n- aws_vpc.main (create)",
						RawPlanOutput:     "+ resource \"aws_vpc\" \"main\"",
					}},
				},
			},
		},
	}

	rendered := SummaryReportCLI(report)
	for _, wanted := range []string{
		"Terraform Plan Summary",
		"2 modules: 1 with changes, 1 no changes, 0 failed",
		"Summary",
		"• warn Cost Estimation: 1 module added cost",
		"Environment: `prod`",
		"svc/prod/eu/vpc (+1)",
		"Resources",
		`    + resource "aws_vpc" "main"`,
		"┌",
	} {
		if !strings.Contains(rendered, wanted) {
			t.Fatalf("rendered output missing %q:\n%s", wanted, rendered)
		}
	}
}
