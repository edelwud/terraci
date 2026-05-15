package plugin

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
)

func TestReportRegistry_PublishAndGet(t *testing.T) {
	r := NewReportRegistry()

	report := &ci.Report{Producer: "feature_a", Title: "Feature Estimation", Status: ci.ReportStatusPass}
	r.Publish(report)

	got, ok := r.Get("feature_a")
	if !ok {
		t.Fatal("expected to find report")
	}
	if got.Title != "Feature Estimation" {
		t.Errorf("Title = %q, want Feature Estimation", got.Title)
	}
}

func TestReportRegistry_GetMissing(t *testing.T) {
	r := NewReportRegistry()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestReportRegistry_PublishOverwrite(t *testing.T) {
	r := NewReportRegistry()

	r.Publish(&ci.Report{Producer: "feature_a", Title: "v1"})
	r.Publish(&ci.Report{Producer: "feature_a", Title: "v2"})

	got, _ := r.Get("feature_a")
	if got.Title != "v2" {
		t.Errorf("expected overwrite, got Title = %q", got.Title)
	}
}

func TestReportRegistry_All(t *testing.T) {
	r := NewReportRegistry()

	r.Publish(&ci.Report{Producer: "report_a", Title: "Report A"})
	r.Publish(&ci.Report{Producer: "report_b", Title: "Report B"})

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d reports, want 2", len(all))
	}

	titles := map[string]bool{}
	for _, rep := range all {
		titles[rep.Title] = true
	}
	if !titles["Report A"] || !titles["Report B"] {
		t.Error("expected both reports")
	}
}

func TestReportRegistry_AllEmpty(t *testing.T) {
	r := NewReportRegistry()

	all := r.All()
	if len(all) != 0 {
		t.Errorf("All() on empty registry returned %d, want 0", len(all))
	}
}

func TestReportRegistry_DefensiveCopies(t *testing.T) {
	r := NewReportRegistry()

	report := &ci.Report{
		Producer: "report_b",
		Title:    "Report B",
		Sections: []ci.ReportSection{citest.MustEncodeRenderSection(
			"Findings",
			"",
			ci.ReportStatusWarn,
			ci.RenderTableBlock("", []string{"Module", "Severity", "Message"}, [][]string{{"app", "warn", "original"}}),
		)},
	}
	r.Publish(report)

	report.Title = "mutated"
	report.Sections[0].Payload[0] = '{'

	got, ok := r.Get("report_b")
	if !ok {
		t.Fatal("expected to find report")
	}
	if got.Title != "Report B" {
		t.Fatalf("stored report title = %q, want Report B", got.Title)
	}
	rendered, err := ci.DecodeSection[ci.RenderSection](got.Sections[0])
	if err != nil {
		t.Fatalf("decode original render section: %v", err)
	}
	if rendered.Blocks[0].Table.Rows[0][2] != "original" {
		t.Fatalf("stored row was mutated: %q", rendered.Blocks[0].Table.Rows[0][2])
	}

	got.Sections[0].Payload[0] = '{'
	gotAgain, _ := r.Get("report_b")
	renderedAgain, err := ci.DecodeSection[ci.RenderSection](gotAgain.Sections[0])
	if err != nil {
		t.Fatalf("decode again: %v", err)
	}
	if renderedAgain.Blocks[0].Table.Rows[0][2] != "original" {
		t.Fatalf("Get returned shared report state: %q", renderedAgain.Blocks[0].Table.Rows[0][2])
	}
}
