// Package citest hosts test-only helpers for pkg/ci consumers and producers.
package citest

import "github.com/edelwud/terraci/pkg/ci"

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
