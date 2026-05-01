package ci

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Validate verifies the persisted report artifact contract.
func (r *Report) Validate() error {
	if r == nil {
		return errors.New("ci report is nil")
	}
	if strings.TrimSpace(r.Plugin) == "" {
		return errors.New("ci report plugin is required")
	}
	if strings.ContainsAny(r.Plugin, `/\`) {
		return fmt.Errorf("ci report plugin %q is not a safe artifact name", r.Plugin)
	}
	if strings.TrimSpace(r.Title) == "" {
		return errors.New("ci report title is required")
	}
	if !r.Status.Valid() {
		return fmt.Errorf("ci report status %q is invalid", r.Status)
	}
	for i := range r.Sections {
		if err := validateReportSection(r.Sections[i]); err != nil {
			return fmt.Errorf("ci report section %d: %w", i, err)
		}
	}
	return nil
}

// MarshalJSON validates the section union before encoding.
func (s ReportSection) MarshalJSON() ([]byte, error) {
	if err := validateReportSection(s); err != nil {
		return nil, err
	}

	type alias ReportSection
	return json.Marshal(alias(s))
}

// UnmarshalJSON decodes and validates the section union.
func (s *ReportSection) UnmarshalJSON(data []byte) error {
	type alias ReportSection
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	section := ReportSection(decoded)
	if err := validateReportSection(section); err != nil {
		return err
	}

	*s = section
	return nil
}

func validateReportSection(s ReportSection) error {
	payloads := []reportSectionPayload{
		{name: "overview", present: s.Overview != nil},
		{name: "module_table", present: s.ModuleTable != nil},
		{name: "findings", present: s.Findings != nil},
		{name: "dependency_updates", present: s.DependencyUpdates != nil},
		{name: "payload", present: len(s.Payload) > 0},
	}

	expectedPayload, err := expectedReportSectionPayload(s.Kind)
	if err != nil {
		return err
	}

	matched := 0
	for i := range payloads {
		payload := payloads[i]
		if !payload.present {
			continue
		}
		matched++
		if payload.name != expectedPayload {
			return fmt.Errorf("report section %q has unexpected %s payload", s.Kind, payload.name)
		}
	}

	if matched == 0 {
		return fmt.Errorf("report section %q is missing %s payload", s.Kind, expectedPayload)
	}
	if matched != 1 {
		return fmt.Errorf("report section %q must contain exactly one payload", s.Kind)
	}
	return nil
}

func expectedReportSectionPayload(kind ReportSectionKind) (string, error) {
	if strings.TrimSpace(string(kind)) == "" {
		return "", errors.New("report section kind is required")
	}
	switch kind {
	case ReportSectionKindOverview:
		return "overview", nil
	case ReportSectionKindModuleTable:
		return "module_table", nil
	case ReportSectionKindFindings:
		return "findings", nil
	case ReportSectionKindDependencyUpdates:
		return "dependency_updates", nil
	default:
		return "payload", nil
	}
}
