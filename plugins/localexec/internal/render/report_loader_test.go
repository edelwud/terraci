package render

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
)

func TestSummaryReportLoaderNoops(t *testing.T) {
	t.Parallel()

	loader := NewSummaryReportLoader(ci.NewMemoryReportStore(), t.TempDir(), nil)
	if err := loader.Reset(context.Background()); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	report, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if report != nil {
		t.Fatalf("Load() report = %#v, want nil", report)
	}
}

func TestSummaryReportLoaderBuildsReportFromStore(t *testing.T) {
	t.Parallel()

	store := ci.NewMemoryReportStore()
	store.Publish(citest.MustRenderedReport(ci.RenderedReportOptions{
		Producer: "cost",
		Title:    "Cost",
		Status:   ci.ReportStatusWarn,
		Summary:  "cost changed",
		Sections: []ci.RenderedSectionOptions{{
			Title:   "Cost",
			Summary: "cost changed",
			Blocks:  []ci.RenderBlock{ci.RenderTextBlock("cost body")},
		}},
	}))

	loader := NewSummaryReportLoader(store, t.TempDir(), nil)
	report, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if report == nil {
		t.Fatal("Load() report = nil, want aggregate report")
	}
	if report.Producer != summaryReportProducer || report.Status != ci.ReportStatusWarn {
		t.Fatalf("aggregate report = %#v, want summary warning report", report)
	}
	if len(report.Sections) != 1 {
		t.Fatalf("sections len = %d, want 1", len(report.Sections))
	}
}

func TestSelectCurrentReportsSkipsStaleFingerprint(t *testing.T) {
	t.Parallel()

	collection := &ci.PlanResultCollection{Results: []ci.PlanResult{{
		ModuleID:   "svc/prod/us/vpc",
		ModulePath: "svc/prod/us/vpc",
		Status:     ci.PlanStatusChanges,
		Summary:    "+1",
	}}}
	fresh := citest.MustRenderedReport(ci.RenderedReportOptions{
		Producer: "fresh",
		Title:    "Fresh",
		Status:   ci.ReportStatusPass,
		Artifact: ci.NewArtifactContext(ci.ArtifactContextOptions{
			PlanResultsFingerprint: collection.Fingerprint(),
		}),
		Sections: []ci.RenderedSectionOptions{{
			Title:  "Fresh",
			Blocks: []ci.RenderBlock{ci.RenderTextBlock("fresh")},
		}},
	})
	stale := citest.MustRenderedReport(ci.RenderedReportOptions{
		Producer: "stale",
		Title:    "Stale",
		Status:   ci.ReportStatusPass,
		Artifact: ci.NewArtifactContext(ci.ArtifactContextOptions{
			PlanResultsFingerprint: "old",
		}),
		Sections: []ci.RenderedSectionOptions{{
			Title:  "Stale",
			Blocks: []ci.RenderBlock{ci.RenderTextBlock("stale")},
		}},
	})

	selected := selectCurrentReports(collection, []*ci.Report{stale, fresh})
	if len(selected) != 1 || selected[0].Producer != "fresh" {
		t.Fatalf("selected = %#v, want only fresh report", selected)
	}
}
