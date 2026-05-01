package render

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestRenderSummaryReportCLI_RendersStructuredSections(t *testing.T) {
	t.Parallel()

	report := &ci.Report{
		Producer: "summary",
		Title:    "Terraform Plan Summary",
		Summary:  "2 modules: 1 with changes, 1 no changes, 0 failed",
		Sections: []ci.ReportSection{
			ci.MustEncodeSection(
				ci.ReportSectionKindOverview,
				"Summary",
				"2 modules: 1 with changes, 1 no changes, 0 failed",
				ci.ReportStatusWarn,
				ci.OverviewSection{
					PlanStats: ci.SummaryPlanStats{Total: 2, Changes: 1, NoChanges: 1, Success: 2},
					Reports: []ci.SummaryReportOverview{
						{Kind: "cost_changes", Title: "Cost Estimation", Status: ci.ReportStatusWarn, Summary: "1 module added cost"},
					},
				},
			),
			ci.MustEncodeSection(
				ci.ReportSectionKindModuleTable,
				"Environment: `prod`",
				"1 actionable modules",
				ci.ReportStatusWarn,
				ci.ModuleTableSection{
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
			),
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
