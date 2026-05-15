package ci

import (
	"encoding/json"
	"fmt"
)

// encodeSection builds a ReportSection by JSON-encoding the body.
func encodeSection[T any](kind ReportSectionKind, title, sectionSummary string, status ReportStatus, body T) (ReportSection, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return ReportSection{}, fmt.Errorf("encode %s section payload: %w", kind, err)
	}
	return ReportSection{
		Kind:           kind,
		Title:          title,
		Status:         status,
		SectionSummary: sectionSummary,
		Payload:        data,
	}, nil
}

// DecodeSection JSON-decodes the section payload into T. Consumers select the
// expected payload type based on Section.Kind.
func DecodeSection[T any](section ReportSection) (T, error) {
	var out T
	if len(section.Payload) == 0 {
		return out, fmt.Errorf("section %q has empty payload", section.Kind)
	}
	if err := json.Unmarshal(section.Payload, &out); err != nil {
		return out, fmt.Errorf("decode %s section payload: %w", section.Kind, err)
	}
	return out, nil
}
