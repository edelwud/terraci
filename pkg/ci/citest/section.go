// Package citest hosts test-only helpers for pkg/ci consumers and producers.
package citest

import (
	"encoding/json"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

// ReportRenderer renders a report in tests. It lets plugin-side tests pass
// internal renderer functions without pkg/ci depending on plugin internals.
type ReportRenderer func(*ci.Report) (string, error)

// RenderedReportContract describes the canonical persisted-report expectations
// shared by producer plugin tests.
type RenderedReportContract struct {
	Producer    string
	Status      ci.ReportStatus
	Fingerprint string
	Renderers   []ReportRenderer
}

// MustRenderedSection is the panic-on-error variant of ci.NewRenderedSection,
// intended only for tests where the body is statically valid.
func MustRenderedSection(title, sectionSummary string, status ci.ReportStatus, blocks ...ci.RenderBlock) ci.ReportSection {
	section, err := ci.NewRenderedSection(ci.RenderedSectionOptions{
		Title:   title,
		Summary: sectionSummary,
		Status:  status,
		Blocks:  blocks,
	})
	if err != nil {
		panic(err)
	}
	return section
}

// MustRenderedReport is the panic-on-error variant of ci.NewRenderedReport,
// intended only for tests where the report is statically valid.
func MustRenderedReport(opts ci.RenderedReportOptions) *ci.Report {
	report, err := ci.NewRenderedReport(opts)
	if err != nil {
		panic(err)
	}
	return report
}

// MustReportSectionJSON decodes a persisted-section JSON fixture. It is useful
// for malformed artifact tests now that ReportSection is a value object.
func MustReportSectionJSON(raw string) ci.ReportSection {
	var section ci.ReportSection
	if err := json.Unmarshal([]byte(raw), &section); err != nil {
		panic(err)
	}
	return section
}

// AssertRenderedReportContract verifies the canonical producer report contract:
// reports validate, all sections decode through ci.DecodeRenderSection, optional
// provenance expectations match, and configured renderers can consume the report.
func AssertRenderedReportContract(tb testing.TB, report *ci.Report, contract RenderedReportContract) {
	tb.Helper()
	if report == nil {
		tb.Fatal("report = nil")
		return
	}
	if err := report.Validate(); err != nil {
		tb.Fatalf("report.Validate() error = %v", err)
	}
	if contract.Producer != "" && report.Producer != contract.Producer {
		tb.Fatalf("Producer = %q, want %q", report.Producer, contract.Producer)
	}
	if contract.Status != "" && report.Status != contract.Status {
		tb.Fatalf("Status = %q, want %q", report.Status, contract.Status)
	}
	if contract.Fingerprint != "" {
		if report.Provenance == nil {
			tb.Fatalf("Provenance = nil, want fingerprint %q", contract.Fingerprint)
		}
		if got := report.Provenance.PlanResultsFingerprint; got != contract.Fingerprint {
			tb.Fatalf("Provenance.PlanResultsFingerprint = %q, want %q", got, contract.Fingerprint)
		}
	}
	for i, section := range report.Sections {
		if section.Kind() != ci.ReportSectionKindRendered {
			tb.Fatalf("section %d kind = %q, want %q", i, section.Kind(), ci.ReportSectionKindRendered)
		}
		if _, err := ci.DecodeRenderSection(section); err != nil {
			tb.Fatalf("DecodeRenderSection(%d) error = %v", i, err)
		}
	}
	for i, render := range contract.Renderers {
		if _, err := render(report); err != nil {
			tb.Fatalf("renderer %d error = %v", i, err)
		}
	}
}
