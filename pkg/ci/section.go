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
		kind:           kind,
		title:          title,
		status:         status,
		sectionSummary: sectionSummary,
		payload:        data,
	}, nil
}

// decodeSection JSON-decodes the section payload into T. Consumers select the
// expected payload type through typed public helpers such as DecodeRenderSection.
func decodeSection[T any](section ReportSection) (T, error) {
	var out T
	if len(section.payload) == 0 {
		return out, fmt.Errorf("section %q has empty payload", section.kind)
	}
	if err := json.Unmarshal(section.payload, &out); err != nil {
		return out, fmt.Errorf("decode %s section payload: %w", section.kind, err)
	}
	return out, nil
}
