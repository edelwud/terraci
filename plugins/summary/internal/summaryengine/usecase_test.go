package summaryengine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestRun_SkipsReportWithMismatchedFingerprint(t *testing.T) {
	t.Parallel()

	collection := &ci.PlanResultCollection{
		GeneratedAt: time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		Results: []ci.PlanResult{{
			ModuleID: "svc/prod/us/vpc",
			Status:   ci.PlanStatusChanges,
			Summary:  "+1",
		}},
	}
	currentFingerprint := collection.Fingerprint()
	fresh := mustRenderedReport(t, "fresh", "Fresh Report", currentFingerprint)
	stale := mustRenderedReport(t, "stale", "Stale Report", "old-fingerprint")

	result, err := Run(context.Background(), Runtime{
		Config: Config{},
		PlanScanner: fakePlanScanner{
			collection: collection,
		},
		ReportStore: testReportStore(stale, fresh),
		ProviderResolver: func() (Provider, error) {
			return fakeProvider{service: fakeCommentService{enabled: false}}, nil
		},
	}, Request{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(result.Reports) != 1 || result.Reports[0].Producer != "fresh" {
		t.Fatalf("Reports = %#v, want only fresh report", result.Reports)
	}
	if len(result.ReportWarnings) != 1 || !strings.Contains(result.ReportWarnings[0], "stale") {
		t.Fatalf("ReportWarnings = %v, want stale warning", result.ReportWarnings)
	}
	if !strings.Contains(result.Body, "Fresh Report") {
		t.Fatalf("Body missing fresh report:\n%s", result.Body)
	}
	if strings.Contains(result.Body, "Stale Report") {
		t.Fatalf("Body contains stale report:\n%s", result.Body)
	}
}

func TestRun_ReportWithoutFingerprintRendersWithoutWarning(t *testing.T) {
	t.Parallel()

	collection := &ci.PlanResultCollection{
		GeneratedAt: time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC),
		Results:     []ci.PlanResult{{ModuleID: "svc/prod/us/vpc", Status: ci.PlanStatusChanges, Summary: "+1"}},
	}
	report := mustRenderedReport(t, "legacy", "Legacy Report", "")

	result, err := Run(context.Background(), Runtime{
		PlanScanner: fakePlanScanner{collection: collection},
		ReportStore: testReportStore(report),
		ProviderResolver: func() (Provider, error) {
			return fakeProvider{service: fakeCommentService{enabled: false}}, nil
		},
	}, Request{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(result.ReportWarnings) != 0 {
		t.Fatalf("ReportWarnings = %v, want none for degraded provenance mode", result.ReportWarnings)
	}
	if !strings.Contains(result.Body, "Legacy Report") {
		t.Fatalf("Body missing legacy report:\n%s", result.Body)
	}
}

func mustRenderedReport(t *testing.T, producer, title, fingerprint string) *ci.Report {
	t.Helper()
	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: producer,
		Title:    title,
		Status:   ci.ReportStatusWarn,
		Summary:  "summary",
		Artifact: ci.NewArtifactContext(ci.ArtifactContextOptions{
			PlanResultsFingerprint: fingerprint,
		}),
		Sections: []ci.RenderedSectionOptions{{
			Title:   title,
			Summary: "summary",
			Blocks:  []ci.RenderBlock{ci.NewTextBlock(ci.RenderText("body"))},
		}},
	})
	if err != nil {
		t.Fatalf("NewRenderedReport() error = %v", err)
	}
	return report
}

func testReportStore(reports ...*ci.Report) ci.ReportStore {
	store := ci.NewMemoryReportStore()
	for _, report := range reports {
		store.Publish(report)
	}
	return store
}

type fakePlanScanner struct {
	collection *ci.PlanResultCollection
	err        error
}

func (s fakePlanScanner) ScanPlanResults(string, []string) (*ci.PlanResultCollection, error) {
	return s.collection, s.err
}

type fakeProvider struct {
	service ci.CommentService
}

func (p fakeProvider) CommitSHA() string  { return "" }
func (p fakeProvider) PipelineID() string { return "" }
func (p fakeProvider) CommentService() (ci.CommentService, bool) {
	return p.service, p.service != nil
}

type fakeCommentService struct {
	enabled bool
}

func (s fakeCommentService) IsEnabled() bool {
	return s.enabled
}

func (s fakeCommentService) UpsertComment(context.Context, string) error {
	return nil
}
