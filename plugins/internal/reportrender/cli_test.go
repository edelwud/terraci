package reportrender

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
)

func TestCLIReport_Golden(t *testing.T) {
	t.Parallel()

	report := &ci.Report{
		Producer: "policy",
		Title:    "Policy Check",
		Status:   ci.ReportStatusWarn,
		Summary:  "1 finding",
		Sections: []ci.ReportSection{citest.MustRenderedSection(
			"Findings",
			"1 finding",
			ci.ReportStatusWarn,
			ci.RenderTextBlock("Review these changes"),
			ci.RenderTableBlock("Modules", []string{"Module", "Status"}, [][]string{{"svc/prod/vpc", "needs review"}}),
		)},
	}

	rendered, err := CLIReport(report)
	if err != nil {
		t.Fatalf("CLIReport() error = %v", err)
	}

	want := strings.TrimSpace(`
Policy Check
════════════
1 finding

Findings
────────
Warning - 1 finding

Review these changes

Modules
───────
┌──────────────┬──────────────┐
│ Module       │ Status       │
├──────────────┼──────────────┤
│ svc/prod/vpc │ needs review │
└──────────────┴──────────────┘
`)
	if rendered != want {
		t.Fatalf("CLIReport() mismatch\nwant:\n%s\n\ngot:\n%s", want, rendered)
	}
	if strings.Contains(rendered, "warn") {
		t.Fatalf("CLIReport() contains raw warn label:\n%s", rendered)
	}
}

func TestCLIReport_RejectsInvalidRenderedPayload(t *testing.T) {
	t.Parallel()

	report := &ci.Report{
		Producer: "policy",
		Title:    "Policy Check",
		Status:   ci.ReportStatusWarn,
		Sections: []ci.ReportSection{citest.MustReportSectionJSON(`{
			"kind": "rendered",
			"title": "Findings",
			"status": "warn",
			"payload": {
				"blocks": [{
					"kind": "table",
					"table": {
						"columns": ["Module"],
						"rows": [["app", "extra"]]
					}
				}]
			}
		}`)},
	}

	_, err := CLIReport(report)
	if err == nil {
		t.Fatal("CLIReport() error = nil, want invalid rendered payload error")
	}
	if !strings.Contains(err.Error(), "table row 0 has 2 cells for 1 columns") {
		t.Fatalf("CLIReport() error = %q, want invalid table shape", err.Error())
	}
}
