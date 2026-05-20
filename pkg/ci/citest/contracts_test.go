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
			Blocks: []ci.RenderBlock{ci.RenderTextBlock("ok")},
		}},
	})

	AssertRenderedReportContract(t, report, RenderedReportContract{
		Producer:    "contract",
		Status:      ci.ReportStatusWarn,
		Fingerprint: "fingerprint",
		Renderers: []ReportRenderer{func(r *ci.Report) (string, error) {
			if r.Producer != "contract" {
				t.Fatalf("renderer saw producer %q, want contract", r.Producer)
			}
			return strings.ToUpper(r.Summary), nil
		}},
	})
}

func TestAssertPublishArtifactsContract(t *testing.T) {
	report := MustRenderedReport(ci.RenderedReportOptions{
		Producer: "contract",
		Title:    "Contract Report",
		Status:   ci.ReportStatusPass,
		Sections: []ci.RenderedSectionOptions{{
			Title:  "Details",
			Blocks: []ci.RenderBlock{ci.RenderTextBlock("ok")},
		}},
	})

	AssertPublishArtifactsContract(t, "contract", report)
}
