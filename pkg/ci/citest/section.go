// Package citest hosts test-only helpers for pkg/ci consumers and producers.
package citest

import "github.com/edelwud/terraci/pkg/ci"

// MustEncodeRenderSection is the panic-on-error variant of
// ci.EncodeRenderSection, intended only for tests where the body is statically
// known to be marshalable.
func MustEncodeRenderSection(title, sectionSummary string, status ci.ReportStatus, blocks ...ci.RenderBlock) ci.ReportSection {
	section, err := ci.EncodeRenderSection(title, sectionSummary, status, blocks...)
	if err != nil {
		panic(err)
	}
	return section
}
