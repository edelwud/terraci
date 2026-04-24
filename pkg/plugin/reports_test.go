package plugin

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestReportRegistry_PublishAndGet(t *testing.T) {
	r := NewReportRegistry()

	report := &ci.Report{Plugin: "cost", Title: "Cost Estimation", Status: ci.ReportStatusPass}
	r.Publish(report)

	got, ok := r.Get("cost")
	if !ok {
		t.Fatal("expected to find report")
	}
	if got.Title != "Cost Estimation" {
		t.Errorf("Title = %q, want Cost Estimation", got.Title)
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

	r.Publish(&ci.Report{Plugin: "cost", Title: "v1"})
	r.Publish(&ci.Report{Plugin: "cost", Title: "v2"})

	got, _ := r.Get("cost")
	if got.Title != "v2" {
		t.Errorf("expected overwrite, got Title = %q", got.Title)
	}
}

func TestReportRegistry_All(t *testing.T) {
	r := NewReportRegistry()

	r.Publish(&ci.Report{Plugin: "cost", Title: "Cost"})
	r.Publish(&ci.Report{Plugin: "policy", Title: "Policy"})

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d reports, want 2", len(all))
	}

	titles := map[string]bool{}
	for _, rep := range all {
		titles[rep.Title] = true
	}
	if !titles["Cost"] || !titles["Policy"] {
		t.Error("expected both Cost and Policy reports")
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
		Plugin: "policy",
		Title:  "Policy",
		Sections: []ci.ReportSection{{
			Kind:  ci.ReportSectionKindFindings,
			Title: "Findings",
			Findings: &ci.FindingsSection{
				Rows: []ci.FindingRow{{
					ModulePath: "app",
					Status:     ci.FindingRowStatusWarn,
					Findings: []ci.Finding{{
						Severity: ci.FindingSeverityWarn,
						Message:  "original",
					}},
				}},
			},
		}},
	}
	r.Publish(report)

	report.Title = "mutated"
	report.Sections[0].Findings.Rows[0].Findings[0].Message = "mutated"

	got, ok := r.Get("policy")
	if !ok {
		t.Fatal("expected to find report")
	}
	if got.Title != "Policy" {
		t.Fatalf("stored report title = %q, want Policy", got.Title)
	}
	if got.Sections[0].Findings.Rows[0].Findings[0].Message != "original" {
		t.Fatalf("stored finding was mutated: %q", got.Sections[0].Findings.Rows[0].Findings[0].Message)
	}

	got.Sections[0].Findings.Rows[0].Findings[0].Message = "mutated after get"
	gotAgain, _ := r.Get("policy")
	if gotAgain.Sections[0].Findings.Rows[0].Findings[0].Message != "original" {
		t.Fatalf("Get returned shared report state: %q", gotAgain.Sections[0].Findings.Rows[0].Findings[0].Message)
	}
}
