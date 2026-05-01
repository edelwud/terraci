package render

import (
	"encoding/json"

	"github.com/edelwud/terraci/pkg/ci"
)

const costChangesSectionKind ci.ReportSectionKind = "cost_changes"

type costChangesPayload struct {
	Totals costTotals      `json:"totals"`
	Rows   []costChangeRow `json:"rows,omitempty"`
}

type costTotals struct {
	Before float64 `json:"before,omitempty"`
	After  float64 `json:"after,omitempty"`
	Diff   float64 `json:"diff,omitempty"`
}

type costChangeRow struct {
	ModulePath string  `json:"module_path"`
	Before     float64 `json:"before,omitempty"`
	After      float64 `json:"after,omitempty"`
	Diff       float64 `json:"diff,omitempty"`
	Error      string  `json:"error,omitempty"`
	Notes      string  `json:"notes,omitempty"`
}

func decodeCostChangesPayload(section ci.ReportSection) (costChangesPayload, bool) {
	if section.Kind != costChangesSectionKind || len(section.Payload) == 0 {
		return costChangesPayload{}, false
	}
	var payload costChangesPayload
	if err := json.Unmarshal(section.Payload, &payload); err != nil {
		return costChangesPayload{}, false
	}
	return payload, true
}
