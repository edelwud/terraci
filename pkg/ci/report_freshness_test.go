package ci

import (
	"strings"
	"testing"
)

func TestSelectCurrentReports_SelectsCurrentAndDegradedReports(t *testing.T) {
	t.Parallel()

	collection := testPlanResultCollection()
	current := renderedFreshnessReport(t, "cost", collection.Fingerprint())
	degraded := renderedFreshnessReport(t, "tfupdate", "")
	stale := renderedFreshnessReport(t, "policy", "old")

	selection := SelectCurrentReports(collection, NewReportCollection(stale, nil, degraded, current), ReportSelectionOptions{
		Consumer: "summary",
	})
	if got := producers(selection.Reports()); strings.Join(got, ",") != "cost,tfupdate" {
		t.Fatalf("selected producers = %v, want [cost tfupdate]", got)
	}
	messages := selection.Diagnostics().Messages()
	if len(messages) != 1 {
		t.Fatalf("diagnostics = %v, want one stale warning", messages)
	}
	if !strings.Contains(messages[0], `summary report "policy" skipped`) {
		t.Fatalf("diagnostic = %q, want policy stale warning", messages[0])
	}
}

func TestSelectCurrentReports_ExcludesProducersAndDedupesDeterministically(t *testing.T) {
	t.Parallel()

	collection := testPlanResultCollection()
	older := renderedFreshnessReportWithTitle(t, "cost", "older", collection.Fingerprint())
	newer := renderedFreshnessReportWithTitle(t, "cost", "newer", collection.Fingerprint())

	selection := SelectCurrentReports(collection, NewReportCollection(
		renderedFreshnessReport(t, "summary", collection.Fingerprint()),
		renderedFreshnessReport(t, "tfupdate", ""),
		older,
		newer,
	), ReportSelectionOptions{
		ExcludeProducers: []string{"summary"},
	})

	reports := selection.Reports()
	if got := producers(reports); strings.Join(got, ",") != "cost,tfupdate" {
		t.Fatalf("selected producers = %v, want [cost tfupdate]", got)
	}
	if reports[0].Title() != "newer" {
		t.Fatalf("deduped cost title = %q, want newer", reports[0].Title())
	}
}

func TestSelectCurrentReports_ReturnsDefensiveCopies(t *testing.T) {
	t.Parallel()

	report := renderedFreshnessReport(t, "cost", "")
	selection := SelectCurrentReports(nil, NewReportCollection(report), ReportSelectionOptions{})
	reports := selection.Reports()
	if len(reports) != 1 {
		t.Fatalf("reports len = %d, want 1", len(reports))
	}
	reports[0].title = "mutated"
	if report.Title() == "mutated" {
		t.Fatal("SelectCurrentReports returned original report pointer")
	}
}

func TestEvaluateReportFreshness_Statuses(t *testing.T) {
	t.Parallel()

	collection := testPlanResultCollection()
	tests := []struct {
		name   string
		report *Report
		want   ReportFreshnessStatus
	}{
		{name: "nil report", want: ReportFreshnessDegraded},
		{name: "nil provenance", report: &Report{producer: "manual"}, want: ReportFreshnessDegraded},
		{name: "empty fingerprint", report: renderedFreshnessReport(t, "manual", ""), want: ReportFreshnessDegraded},
		{name: "current", report: renderedFreshnessReport(t, "cost", collection.Fingerprint()), want: ReportFreshnessCurrent},
		{name: "stale", report: renderedFreshnessReport(t, "policy", "old"), want: ReportFreshnessStale},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EvaluateReportFreshness(collection, tt.report, "summary")
			if got.Status() != tt.want {
				t.Fatalf("Status = %q, want %q", got.Status(), tt.want)
			}
		})
	}
}

func renderedFreshnessReport(t *testing.T, producer, fingerprint string) *Report {
	return renderedFreshnessReportWithTitle(t, producer, producer+" report", fingerprint)
}

func renderedFreshnessReportWithTitle(t *testing.T, producer, title, fingerprint string) *Report {
	t.Helper()

	report, err := NewRenderedReport(RenderedReportOptions{
		Producer: producer,
		Title:    title,
		Status:   ReportStatusPass,
		Artifact: NewArtifactContext(ArtifactContextOptions{
			PlanResultsFingerprint: fingerprint,
		}),
		Sections: []RenderedSectionOptions{{
			Title:  producer,
			Blocks: []RenderBlock{NewTextBlock(RenderText("body"))},
		}},
	})
	if err != nil {
		t.Fatalf("NewRenderedReport(%s): %v", producer, err)
	}
	return report
}

func producers(reports []*Report) []string {
	values := make([]string, 0, len(reports))
	for _, report := range reports {
		values = append(values, report.Producer())
	}
	return values
}
