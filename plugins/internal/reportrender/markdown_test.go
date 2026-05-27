package reportrender

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
)

func TestMarkdownSection_RendersEscapedRenderBlocks(t *testing.T) {
	t.Parallel()

	section, err := ci.NewRenderedSection(ci.RenderedSectionOptions{
		Title:   "Policy|Check",
		Summary: "line one\nline two",
		Status:  ci.ReportStatusWarn,
		Blocks: []ci.RenderBlock{
			ci.NewTableBlock(
				"",
				[]ci.RenderColumn{ci.NewRenderColumn("Module|Path"), ci.NewRenderColumn("Summary"), ci.NewRenderColumn("Delta")},
				[]ci.RenderRow{ci.NewRenderRow(
					ci.RenderModulePath("svc|prod\nvpc"),
					ci.RenderText("+1 | update"),
					ci.RenderMoneyDelta(1.25, ci.RenderMoneyOptions{Unit: ci.RenderMoneyUnitMonth}),
				)},
			),
			ci.NewDetailsBlock("prod <vpc>", "raw | markdown\nsecond line", ""),
		},
	})
	if err != nil {
		t.Fatalf("NewRenderedSection: %v", err)
	}

	rendered, err := MarkdownSection(section)
	if err != nil {
		t.Fatalf("MarkdownSection: %v", err)
	}

	for _, want := range []string{
		"### Warning Policy|Check",
		"line one line two",
		"Module\\|Path",
		"svc\\|prod<br>vpc",
		"+1 \\| update",
		"+$1.25/mo",
		"<summary>prod &lt;vpc&gt;</summary>",
		"raw | markdown\nsecond line",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered markdown missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "**Status:**") {
		t.Fatalf("rendered markdown duplicated status line:\n%s", rendered)
	}
	if strings.Contains(rendered, "### warn ") {
		t.Fatalf("rendered markdown contains raw warn label:\n%s", rendered)
	}
}

func TestMarkdownReport_RejectsNonRenderedSections(t *testing.T) {
	t.Parallel()

	report := &ci.Report{
		Producer: "custom",
		Title:    "Custom",
		Status:   ci.ReportStatusWarn,
		Sections: []ci.ReportSection{citest.MustReportSectionJSON(`{"kind":"domain_specific","payload":{}}`)},
	}

	_, err := MarkdownReport(report)
	if err == nil {
		t.Fatal("MarkdownReport() error = nil, want non-rendered section error")
	}
	if !strings.Contains(err.Error(), `is not "rendered"`) {
		t.Fatalf("MarkdownReport() error = %q, want render-ready contract message", err.Error())
	}
}

func TestMarkdownReport_EmptyReportFallback(t *testing.T) {
	t.Parallel()

	report := &ci.Report{
		Producer: "empty",
		Title:    "Empty Report",
		Status:   ci.ReportStatusPass,
		Summary:  "nothing to show",
	}

	rendered, err := MarkdownReport(report)
	if err != nil {
		t.Fatalf("MarkdownReport() error = %v", err)
	}
	for _, want := range []string{"### Passed Empty Report", "nothing to show"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered markdown missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "**Status:**") {
		t.Fatalf("rendered markdown duplicated status line:\n%s", rendered)
	}
}
