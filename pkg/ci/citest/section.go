// Package citest hosts test-only helpers for pkg/ci consumers and producers.
package citest

import "github.com/edelwud/terraci/pkg/ci"

// MustEncodeSection is the panic-on-error variant of ci.EncodeSection, intended
// only for tests where the body is statically known to be marshalable.
//
// Production code must use ci.EncodeSection and propagate the error: a panic
// during a CI run kills the report writer mid-flight, leaving the workspace
// in a half-published state.
func MustEncodeSection[T any](kind ci.ReportSectionKind, title, sectionSummary string, status ci.ReportStatus, body T) ci.ReportSection {
	section, err := ci.EncodeSection(kind, title, sectionSummary, status, body)
	if err != nil {
		panic(err)
	}
	return section
}
