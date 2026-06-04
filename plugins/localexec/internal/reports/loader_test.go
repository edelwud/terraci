package reports

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/planresults"
)

func TestLoaderNoReports(t *testing.T) {
	t.Parallel()

	loader := NewLoader(ci.NewMemoryReportStore(), t.TempDir(), nil)
	result, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	report := result.Report()
	if report != nil {
		t.Fatalf("Load() report = %#v, want nil", report)
	}
}

func TestLoaderBuildsReportFromStore(t *testing.T) {
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
			Blocks:  []ci.RenderBlock{ci.NewTextBlock(ci.RenderText("cost body"))},
		}},
	}))

	loader := NewLoader(store, t.TempDir(), nil)
	result, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	report := result.Report()
	if report == nil {
		t.Fatal("Load() report = nil, want aggregate report")
	}
	if report.Producer() != SummaryReportProducer || report.Status() != ci.ReportStatusWarn {
		t.Fatalf("aggregate report = %#v, want summary warning report", report)
	}
	if len(report.Sections()) != 1 {
		t.Fatalf("sections len = %d, want 1", len(report.Sections()))
	}
}

func TestLoaderSkipsStaleFingerprint(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	writePlanFixture(t, workDir, "platform/prod/eu-central-1/app")
	collection, err := planresults.Scan(workDir, nil)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	store := ci.NewMemoryReportStore()
	store.Publish(citest.MustRenderedReport(ci.RenderedReportOptions{
		Producer: "stale",
		Title:    "Stale",
		Status:   ci.ReportStatusPass,
		Artifact: ci.NewArtifactContext(ci.ArtifactContextOptions{
			PlanResultsFingerprint: "old",
		}),
		Sections: []ci.RenderedSectionOptions{{
			Title:  "Stale",
			Blocks: []ci.RenderBlock{ci.NewTextBlock(ci.RenderText("stale"))},
		}},
	}))
	store.Publish(citest.MustRenderedReport(ci.RenderedReportOptions{
		Producer: "fresh",
		Title:    "Fresh",
		Status:   ci.ReportStatusPass,
		Artifact: ci.NewArtifactContext(ci.ArtifactContextOptions{
			PlanResultsFingerprint: collection.Fingerprint(),
		}),
		Sections: []ci.RenderedSectionOptions{{
			Title:  "Fresh",
			Blocks: []ci.RenderBlock{ci.NewTextBlock(ci.RenderText("fresh"))},
		}},
	}))

	result, err := NewLoader(store, workDir, nil).Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	report := result.Report()
	if report == nil {
		t.Fatal("Load() report = nil, want aggregate report")
	}
	if result.Diagnostics().Len() != 1 {
		t.Fatalf("diagnostics = %v, want one stale report warning", result.Diagnostics().Messages())
	}
	if report.Summary() != "1 plugin reports" {
		t.Fatalf("Summary = %q, want one selected report", report.Summary())
	}
	sections := report.Sections()
	if len(sections) != 1 || sections[0].Title() != "Fresh" {
		t.Fatalf("sections = %#v, want only fresh report section", sections)
	}
}

func writePlanFixture(t *testing.T, root, modulePath string) {
	t.Helper()

	dir := filepath.Join(root, filepath.FromSlash(modulePath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	plan := `{
		"format_version": "1.0",
		"resource_changes": [{"type": "aws_s3_bucket", "name": "bucket", "change": {"actions": ["create"]}}]
	}`
	if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
}
