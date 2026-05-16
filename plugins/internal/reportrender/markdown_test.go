package reportrender

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestMarkdownSection_RendersEscapedRenderBlocks(t *testing.T) {
	t.Parallel()

	section, err := ci.NewRenderedSection(ci.RenderedSectionOptions{
		Title:   "Policy|Check",
		Summary: "line one\nline two",
		Status:  ci.ReportStatusWarn,
		Blocks: []ci.RenderBlock{
			ci.RenderTableBlock(
				"",
				[]string{"Module|Path", "Summary"},
				[][]string{{"svc|prod\nvpc", "+1 | update"}},
			),
			ci.RenderDetailsBlock("prod <vpc>", "raw | markdown\nsecond line", ""),
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
		"### warn Policy|Check",
		"line one line two",
		"Module\\|Path",
		"svc\\|prod<br>vpc",
		"+1 \\| update",
		"<summary>prod &lt;vpc&gt;</summary>",
		"raw | markdown\nsecond line",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered markdown missing %q:\n%s", want, rendered)
		}
	}
}

func TestMarkdownReport_RejectsNonRenderedSections(t *testing.T) {
	t.Parallel()

	report := &ci.Report{
		Producer: "custom",
		Title:    "Custom",
		Status:   ci.ReportStatusWarn,
		Sections: []ci.ReportSection{{
			Kind:    "domain_specific",
			Payload: []byte(`{}`),
		}},
	}

	_, err := MarkdownReport(report)
	if err == nil {
		t.Fatal("MarkdownReport() error = nil, want non-rendered section error")
	}
	if !strings.Contains(err.Error(), `is not "rendered"`) {
		t.Fatalf("MarkdownReport() error = %q, want render-ready contract message", err.Error())
	}
}
