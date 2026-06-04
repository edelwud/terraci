package citest

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestAssertRenderedReportContract(t *testing.T) {
	report := MustRenderedReport(ci.RenderedReportOptions{
		Producer: "contract",
		Title:    "Contract Report",
		Status:   ci.ReportStatusWarn,
		Summary:  "contract summary",
		Artifact: ci.NewArtifactContext(ci.ArtifactContextOptions{
			PlanResultsFingerprint: "fingerprint",
		}),
		Sections: []ci.RenderedSectionOptions{{
			Title:  "Details",
			Blocks: []ci.RenderBlock{ci.NewTextBlock(ci.RenderText("ok"))},
		}},
	})

	AssertRenderedReportContract(t, report, RenderedReportContract{
		Producer:              "contract",
		Status:                ci.ReportStatusWarn,
		Fingerprint:           "fingerprint",
		ForbidRawStatusLabels: true,
		RequireSchemaVersion:  true,
		RequireRendererOutput: true,
		Renderers: []ReportRenderer{func(r *ci.Report) (string, error) {
			if r.Producer() != "contract" {
				t.Fatalf("renderer saw producer %q, want contract", r.Producer())
			}
			return "Warning " + strings.ToUpper(r.Summary()), nil
		}},
	})
}

func TestRawStatusLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "human label", output: "Warning Policy Check", want: ""},
		{name: "raw warn", output: "warn Policy Check", want: "warn"},
		{name: "raw fail in markdown", output: "### fail Policy Check", want: "fail"},
		{name: "passed is not pass", output: "1 passed, 0 failed", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := rawStatusLabel(tt.output)
			if tt.want == "" {
				if ok {
					t.Fatalf("rawStatusLabel() = %q, true; want none", got)
				}
				return
			}
			if !ok || got != tt.want {
				t.Fatalf("rawStatusLabel() = %q, %v; want %q, true", got, ok, tt.want)
			}
		})
	}
}

func TestAssertPublishArtifactsContract(t *testing.T) {
	report := MustRenderedReport(ci.RenderedReportOptions{
		Producer: "contract",
		Title:    "Contract Report",
		Status:   ci.ReportStatusPass,
		Sections: []ci.RenderedSectionOptions{{
			Title:  "Details",
			Blocks: []ci.RenderBlock{ci.NewTextBlock(ci.RenderText("ok"))},
		}},
	})

	AssertPublishArtifactsContract(t, "contract", report)
}
